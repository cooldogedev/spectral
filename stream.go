package spectral

import (
	"context"
	"io"
	"slices"
	"sync"

	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

const maxPayloadSize = 128

type splitEntry struct {
	fragments [][]byte
	remaining int
}

type Stream struct {
	conn               *connection
	ctx                context.Context
	cancelFunc         context.CancelFunc
	streamID           protocol.StreamID
	expectedSequenceID uint32
	sequenceID         uint32
	splits             map[uint32]*splitEntry
	ordered            map[uint32][]byte
	buf                []byte
	mu                 sync.Mutex
	cond               *sync.Cond
	once               sync.Once
}

func newStream(conn *connection, streamID protocol.StreamID) *Stream {
	ctx, cancelFunc := context.WithCancel(conn.Context())
	s := &Stream{
		conn:       conn,
		ctx:        ctx,
		cancelFunc: cancelFunc,
		streamID:   streamID,
		splits:     make(map[uint32]*splitEntry),
		ordered:    make(map[uint32][]byte),
		buf:        make([]byte, 0, 1024*1024),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Stream) Read(p []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	default:
	}

	s.mu.Lock()
	for len(s.buf) == 0 && s.buf != nil {
		s.cond.Wait()
	}

	if s.buf == nil {
		s.mu.Unlock()
		return 0, io.EOF
	}

	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	s.mu.Unlock()
	return n, nil
}

func (s *Stream) Write(p []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		return 0, s.ctx.Err()
	default:
	}

	fr := &frame.StreamData{
		StreamID:   s.streamID,
		SequenceID: s.sequenceID,
	}
	if len(p) <= maxPayloadSize {
		fr.Payload = p
		if err := s.conn.write(fr); err != nil {
			return 0, err
		}
	} else if err := s.writeSplit(p, fr); err != nil {
		return 0, err
	}
	s.sequenceID++
	return len(p), nil
}

func (s *Stream) Context() context.Context {
	return s.ctx
}

func (s *Stream) Close() error {
	_ = s.conn.write(&frame.StreamClose{StreamID: s.streamID})
	return s.internalClose()
}

func (s *Stream) writeSplit(p []byte, fr *frame.StreamData) error {
	length := len(p)
	fr.Total = uint32((length + maxPayloadSize - 1) / maxPayloadSize)
	for i := 0; i < int(fr.Total); i++ {
		start := i * maxPayloadSize
		end := start + maxPayloadSize
		if end > length {
			end = length
		}

		fr.Payload = p[start:end]
		fr.Offset = uint32(i)
		if err := s.conn.write(fr); err != nil {
			return err
		}
	}
	return nil
}

func (s *Stream) internalClose() error {
	s.once.Do(func() {
		s.mu.Lock()
		s.buf = s.buf[:0]
		s.buf = nil
		s.mu.Unlock()
		s.cond.Signal()
		s.cancelFunc()
		s.conn.streams.remove(s.streamID)
	})
	return nil
}

func (s *Stream) receive(fr *frame.StreamData) {
	if fr.Total <= 0 {
		s.handleSingle(fr.SequenceID, fr.Payload)
		return
	}

	entry, ok := s.splits[fr.SequenceID]
	if !ok {
		entry = &splitEntry{
			fragments: make([][]byte, fr.Total),
			remaining: int(fr.Total),
		}
		s.splits[fr.SequenceID] = entry
	}
	entry.fragments[fr.Offset] = append([]byte(nil), fr.Payload...)
	entry.remaining--
	if entry.remaining == 0 {
		delete(s.splits, fr.SequenceID)
		s.handleSplit(fr.SequenceID, entry.fragments)
	}
}

func (s *Stream) handleSingle(sequenceID uint32, p []byte) {
	if sequenceID != s.expectedSequenceID {
		s.ordered[sequenceID] = append([]byte(nil), p...)
		return
	}

	s.expectedSequenceID++
	s.mu.Lock()
	if len(s.ordered) > 0 {
		s.order()
	}
	s.buf = append(s.buf, p...)
	s.mu.Unlock()
	s.cond.Signal()
	return
}

func (s *Stream) handleSplit(sequenceID uint32, fragments [][]byte) {
	if sequenceID != s.expectedSequenceID {
		s.ordered[sequenceID] = slices.Concat(fragments...)
		return
	}

	s.expectedSequenceID++
	s.mu.Lock()
	if len(s.ordered) > 0 {
		s.order()
	}

	for _, fragment := range fragments {
		s.buf = append(s.buf, fragment...)
	}
	s.mu.Unlock()
	s.cond.Signal()
	return
}

func (s *Stream) order() {
	for {
		p, ok := s.ordered[s.expectedSequenceID]
		if !ok {
			break
		}
		delete(s.ordered, s.expectedSequenceID)
		s.expectedSequenceID++
		s.buf = append(s.buf, p...)
	}
}
