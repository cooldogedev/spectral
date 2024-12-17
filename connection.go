package spectral

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cooldogedev/spectral/internal/congestion"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/log"
	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	deadlineInf       = time.Duration(math.MaxInt64)
	deadlineImmediate = protocol.TimerGranularity
	inactivityTimeout = time.Second * 30
)

type receivedPacket struct {
	sequenceID uint32
	frames     []frame.Frame
	t          time.Time
}

type Connection interface {
	AcceptStream(ctx context.Context) (*Stream, error)
	OpenStream(ctx context.Context) (*Stream, error)
	CloseWithError(code byte, message string) error
	Context() context.Context
}

var _ Connection = &connection{}

type connection struct {
	conn           *udpConn
	peerAddr       *net.UDPAddr
	connectionID   atomic.Int64
	sequenceID     atomic.Uint32
	ctx            context.Context
	cancelFunc     context.CancelCauseFunc
	sender         *congestion.Sender
	packets        chan *receivedPacket
	ack            *ackQueue
	receiveQueue   *receiveQueue
	retransmission *retransmissionQueue
	sendQueue      *sendQueue
	streams        *streamMap
	discovery      *mtuDiscovery
	rtt            *congestion.RTT
	handler        func(frame.Frame) error
	notify         chan struct{}
	idle           time.Time
	pacingDeadline time.Time
	once           sync.Once
	logger         log.Logger
}

func newConnection(conn *udpConn, peerAddr *net.UDPAddr, connectionID protocol.ConnectionID, parentCtx context.Context, perspective log.Perspective) *connection {
	now := time.Now()
	logger := log.NewLogger(perspective)
	ctx, cancelFunc := context.WithCancelCause(parentCtx)
	c := &connection{
		conn:           conn,
		peerAddr:       peerAddr,
		ctx:            ctx,
		cancelFunc:     cancelFunc,
		sender:         congestion.NewSender(logger, now, protocol.MinPacketSize),
		packets:        make(chan *receivedPacket, 512),
		ack:            newAckQueue(),
		receiveQueue:   newReceiveQueue(),
		retransmission: newRetransmissionQueue(),
		sendQueue:      newSendQueue(),
		streams:        newStreamMap(),
		notify:         make(chan struct{}, 1),
		idle:           now.Add(inactivityTimeout),
		rtt:            congestion.NewRTT(),
		logger:         logger,
	}
	c.connectionID.Store(int64(connectionID))
	c.discovery = newMTUDiscovery(now, func(mtu uint64) {
		c.sender.SetMSS(mtu)
		c.sendQueue.setMSS(mtu)
		c.logger.Log("mtu_update", "new", mtu)
	})
	go c.run(now)
	return c
}

func (c *connection) AcceptStream(_ context.Context) (*Stream, error) {
	return nil, errors.New("method not implemented")
}

func (c *connection) OpenStream(_ context.Context) (*Stream, error) {
	return nil, errors.New("method not implemented")
}

func (c *connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *connection) RemoteAddr() net.Addr {
	return c.peerAddr
}

func (c *connection) CloseWithError(code byte, message string) (err error) {
	_ = c.writeControl(&frame.ConnectionClose{Code: code, Message: message}, true)
	c.logger.Log("connection_close_err", "code", code, "message", message)
	return c.close(message)
}

func (c *connection) Context() context.Context {
	return c.ctx
}

func (c *connection) createStream(streamID protocol.StreamID) (*Stream, error) {
	if c.streams.get(streamID) != nil {
		c.logger.Log("duplicate_stream", "streamID", streamID)
		return nil, fmt.Errorf("stream %v already exists", streamID)
	}
	stream := newStream(streamID, c.ctx, c.sendQueue, c.wake, func() {
		_ = c.writeControl(&frame.StreamClose{StreamID: streamID}, true)
		c.streams.remove(streamID)
	}, c.logger)
	c.streams.add(stream)
	return stream, nil
}

func (c *connection) run(now time.Time) {
	var lastDeadline time.Time
	timer := time.NewTimer(deadlineInf)
	defer func() {
		timer.Stop()
		_ = c.CloseWithError(frame.ConnectionCloseInternal, "")
		c.cleanup()
	}()

runLoop:
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if err := c.maybeSend(now); err != nil {
			break runLoop
		}

		nextDeadline := firstTime(
			c.idle,
			c.ack.next(),
			c.retransmission.next(c.rtt.RTO()),
			c.pacingDeadline,
		)
		if !nextDeadline.IsZero() && nextDeadline.Before(now) {
			now = time.Now()
			if err := c.triggerTimer(now); err != nil {
				break runLoop
			}
			continue
		} else if !nextDeadline.IsZero() && !nextDeadline.Equal(lastDeadline) {
			timer.Reset(nextDeadline.Sub(now))
			lastDeadline = nextDeadline
		}

		select {
		case <-c.ctx.Done():
			return
		case first := <-c.packets:
			now = time.Now()
			c.idle = now.Add(inactivityTimeout)
			if err := c.receive(now, first.t, first.sequenceID, first.frames); err != nil {
				break runLoop
			}

			totalPackets := len(c.packets)
		receiveLoop:
			for i := 0; i < totalPackets; i++ {
				select {
				case pk := <-c.packets:
					if err := c.receive(now, pk.t, pk.sequenceID, pk.frames); err != nil {
						break runLoop
					}

					select {
					case <-c.ctx.Done():
						break runLoop
					default:
					}
				default:
					break receiveLoop
				}
			}
		case <-timer.C:
			now = time.Now()
			if err := c.triggerTimer(now); err != nil {
				break runLoop
			}
		case <-c.notify:
			now = time.Now()
		}
	}
}

func (c *connection) triggerTimer(now time.Time) (err error) {
	if !c.idle.After(now) {
		_ = c.CloseWithError(frame.ConnectionCloseTimeout, "network inactivity")
		return errors.New("network inactivity")
	}

	if err := c.retransmit(now); err != nil {
		return err
	}
	return
}

func (c *connection) receive(now, t time.Time, sequenceID uint32, frames []frame.Frame) (err error) {
	if sequenceID != 0 {
		c.ack.add(t, sequenceID)
		if !c.receiveQueue.add(sequenceID) {
			c.logger.Log("duplicate_receive", "sequenceID", sequenceID)
			return
		}
	}

	for _, fr := range frames {
		if err := c.handler(fr); err != nil {
			return err
		}

		if err := c.handle(now, fr); err != nil {
			return err
		}
	}
	//lint:ignore SA6002 ignore this for now.
	frame.Pool.Put(frames[:0])
	return
}

func (c *connection) handle(now time.Time, fr frame.Frame) (err error) {
	switch fr := fr.(type) {
	case *frame.Acknowledgement:
		for _, r := range fr.Ranges {
			for i := r[0]; i <= r[1]; i++ {
				if entry := c.retransmission.remove(i); entry != nil {
					if i == fr.Max {
						c.rtt.Add(now.Sub(entry.sent), time.Duration(fr.Delay))
					}
					c.sender.OnAck(now, entry.sent, c.rtt, uint64(len(entry.payload)))
				}
			}
		}
	case *frame.ConnectionClose:
		if err := c.close(fr.Message); err != nil {
			return err
		}
	case *frame.StreamData:
		if stream := c.streams.get(fr.StreamID); stream != nil {
			stream.receive(fr.SequenceID, fr.Payload)
		}
	case *frame.StreamClose:
		if stream := c.streams.get(fr.StreamID); stream != nil {
			c.logger.Log("stream_close_request", "streamID", fr.StreamID)
			_ = stream.internalClose("closed by peer")
		}
	case *frame.MTURequest:
		if err := c.writeControl(&frame.MTUResponse{MTU: fr.MTU}, false); err != nil {
			return err
		}
	case *frame.MTUResponse:
		c.discovery.onAck(fr.MTU)
	}
	frame.PutFrame(fr)
	return
}

func (c *connection) writeControl(fr frame.Frame, needsAck bool) (err error) {
	select {
	case <-c.ctx.Done():
		return context.Cause(c.ctx)
	default:
	}

	var sequenceID uint32
	if needsAck {
		sequenceID = c.sequenceID.Add(1)
	}

	pk := frame.Pack(protocol.ConnectionID(c.connectionID.Load()), sequenceID, 1, frame.PackSingle(fr))
	if _, err := c.writeDatagram(pk); err != nil {
		return err
	}

	if needsAck {
		c.retransmission.add(time.Now(), sequenceID, pk)
	}
	return
}

func (c *connection) maybeSend(now time.Time) (err error) {
	if c.conn.mtud && !c.discovery.discovered && c.discovery.sendProbe(now, c.rtt.SRTT()) {
		_ = c.writeControl(&frame.MTURequest{MTU: c.discovery.current}, false)
		c.logger.Log("mtu_probe", "new", c.discovery.current)
	}

	for c.sendQueue.available() {
		wouldBlock, err := c.transmit(now)
		if err != nil {
			return err
		}

		if wouldBlock {
			break
		}
	}

	if !c.sendQueue.available() {
		c.pacingDeadline = time.Time{}
	} else if c.pacingDeadline.IsZero() || now.After(c.pacingDeadline) {
		c.pacingDeadline = now.Add(deadlineImmediate)
	}
	return c.acknowledge(now)
}

func (c *connection) transmit(now time.Time) (wouldBlock bool, err error) {
	available := c.sender.Available()
	if available == 0 {
		c.logger.Log("congestion_block", "window", available)
		return true, nil
	}

	total, p := c.sendQueue.pack(available)
	length := uint64(len(p))
	if total == 0 {
		c.logger.Log("congestion_block", "window", available)
		return true, nil
	}

	if t := c.sender.TimeUntilSend(now, c.rtt, length); !t.IsZero() && t.After(now) {
		c.pacingDeadline = t
		c.logger.Log("pacer_block", "len", length, "window", available)
		return true, nil
	}

	sequenceID := c.sequenceID.Add(1)
	pk := frame.Pack(protocol.ConnectionID(c.connectionID.Load()), sequenceID, total, p)
	c.sendQueue.flush()
	if _, err := c.writeDatagram(pk); err != nil {
		return false, err
	}
	c.sender.OnSend(length)
	c.retransmission.add(now, sequenceID, pk)
	return
}

func (c *connection) acknowledge(now time.Time) (err error) {
	list, maxSequenceID, delay := c.ack.flush(now)
	if len(list) > 0 {
		fr := &frame.Acknowledgement{Delay: delay.Nanoseconds(), Max: maxSequenceID}
		ranges := frame.GenerateAcknowledgementRanges(list)
		for chunk := range slices.Chunk(ranges, 128) {
			fr.Ranges = chunk
			if err := c.writeControl(fr, false); err != nil {
				return err
			}
		}
	}
	return
}

func (c *connection) retransmit(now time.Time) (err error) {
	if pk, t := c.retransmission.shift(now, c.rtt.RTO()); len(pk) > 0 {
		c.sender.OnCongestionEvent(now, t)
		if _, err := c.writeDatagram(pk); err != nil {
			return err
		}
	}
	return
}

func (c *connection) wake() {
	select {
	case c.notify <- struct{}{}:
	default:
	}
}

func (c *connection) writeDatagram(p []byte) (int, error) {
	return c.conn.Write(p, c.peerAddr)
}

func (c *connection) close(message string) (err error) {
	c.once.Do(func() {
		for _, stream := range c.streams.all() {
			_ = stream.internalClose(fmt.Sprintf("closed by connection: %s", message))
		}
		c.cancelFunc(errors.New(message))
		c.logger.Log("connection_close")
		c.logger.Close()
		_ = c.conn.Close()
	})
	return
}

func (c *connection) cleanup() {
	<-c.ctx.Done()
	c.retransmission.clear()
	c.sendQueue.clear()
	c.handler = nil
	c.discovery.mtuIncrease = nil
	clear(c.receiveQueue.queue)
	close(c.packets)
	close(c.notify)
}

func firstTime(idle, ack, retransmission, pacing time.Time) time.Time {
	deadline := idle
	if !ack.IsZero() && ack.Before(deadline) {
		deadline = ack
	}

	if !retransmission.IsZero() && retransmission.Before(deadline) {
		deadline = retransmission
	}

	if !pacing.IsZero() && pacing.Before(deadline) {
		deadline = pacing
	}
	return deadline
}
