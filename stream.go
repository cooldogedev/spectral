package spectral

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/cooldogedev/spectral/internal"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/log"
	"github.com/cooldogedev/spectral/internal/protocol"
)

var streamDataPool = sync.Pool{New: func() any { return &frame.StreamData{} }}

type Stream struct {
	ctx        context.Context
	cancelFunc context.CancelCauseFunc
	streamID   protocol.StreamID
	wake       func()
	closer     func()
	sendQueue  *sendQueue
	frame      *frameQueue
	buffer     *internal.RingBuffer[byte]
	available  chan struct{}
	sequenceID atomic.Uint32
	logger     log.Logger
	mu         sync.Mutex
	once       sync.Once
}

func newStream(streamID protocol.StreamID, parentCtx context.Context, sendQueue *sendQueue, wake func(), closer func(), logger log.Logger) *Stream {
	ctx, cancelFunc := context.WithCancelCause(parentCtx)
	return &Stream{
		ctx:        ctx,
		cancelFunc: cancelFunc,
		streamID:   streamID,
		wake:       wake,
		closer:     closer,
		sendQueue:  sendQueue,
		frame:      newFrameQueue(),
		buffer:     internal.NewRingBuffer[byte](1024 * 1024),
		available:  make(chan struct{}, 1),
		logger:     logger,
	}
}

func (s *Stream) Read(p []byte) (int, error) {
	if n := s.read(p); n > 0 {
		return n, nil
	}

	select {
	case <-s.ctx.Done():
		return 0, context.Cause(s.ctx)
	case <-s.available:
		return s.read(p), nil
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		return 0, context.Cause(s.ctx)
	default:
	}

	mss := int(s.sendQueue.mss()) - 20
	fr := streamDataPool.Get().(*frame.StreamData)
	fr.StreamID = s.streamID
	for payload := range slices.Chunk(p, mss) {
		fr.SequenceID = s.sequenceID.Add(1) - 1
		fr.Payload = payload
		s.sendQueue.add(frame.PackSingle(fr))
	}
	fr.Payload = fr.Payload[:0]
	streamDataPool.Put(fr)
	s.wake()
	return len(p), nil
}

func (s *Stream) Context() context.Context {
	return s.ctx
}

func (s *Stream) Close() error {
	s.logger.Log("stream_close_application", "streamID", s.streamID)
	return s.internalClose("closed by application")
}

func (s *Stream) internalClose(message string) error {
	s.once.Do(func() {
		s.cancelFunc(errors.New(message))
		s.closer()
		s.logger.Log("stream_close", "streamID", s.streamID)
		s.cleanup()
	})
	return nil
}

func (s *Stream) cleanup() {
	<-s.ctx.Done()
	s.mu.Lock()
	s.buffer.Reset()
	s.frame.clear()
	s.mu.Unlock()
}

func (s *Stream) receive(sequenceID uint32, p []byte) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	s.mu.Lock()
	if s.frame.expected == sequenceID && s.buffer.Free() >= len(p) {
		s.frame.expected++
		_, _ = s.buffer.Write(p)
	} else {
		s.frame.enqueue(sequenceID, p)
	}
	s.processFrames()
	s.mu.Unlock()
}

func (s *Stream) processFrames() {
	for {
		entry := s.frame.top()
		if entry == nil || len(entry.payload) > s.buffer.Free() {
			break
		}

		if _, err := s.buffer.Write(entry.payload); err != nil {
			break
		}
		s.frame.dequeue()
	}

	if s.buffer.Len() > 0 {
		select {
		case s.available <- struct{}{}:
		default:
		}
	}
}

func (s *Stream) read(p []byte) (n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buffer.Len() > 0 {
		return s.buffer.Read(p)
	}
	return
}
