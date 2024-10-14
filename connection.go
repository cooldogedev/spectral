package spectral

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cooldogedev/spectral/internal"
	"github.com/cooldogedev/spectral/internal/congestion"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

const inactivityTimeout = time.Second * 30

type Connection interface {
	AcceptStream(ctx context.Context) (*Stream, error)
	OpenStream(ctx context.Context) (*Stream, error)
	CloseWithError(code byte, message string) error
	Context() context.Context
}

var _ Connection = &connection{}

type connection struct {
	conn           *internal.Conn
	connectionID   protocol.ConnectionID
	ctx            context.Context
	cancelFunc     context.CancelCauseFunc
	cc             *congestion.Cubic
	pacer          *congestion.Pacer
	ack            *ackQueue
	receiveQueue   *receiveQueue
	retransmission *retransmissionQueue
	sendQueue      *sendQueue
	streams        *streamMap
	handler        func(frame.Frame) error
	rtt            *internal.RTT
	once           sync.Once
	activity       atomic.Int64
}

func newConnection(conn *internal.Conn, connectionID protocol.ConnectionID, parentCtx context.Context) *connection {
	ctx, cancelFunc := context.WithCancelCause(parentCtx)
	c := &connection{
		conn:           conn,
		connectionID:   connectionID,
		ctx:            ctx,
		cancelFunc:     cancelFunc,
		cc:             congestion.NewCubic(),
		pacer:          congestion.NewPacer(),
		ack:            newAckQueue(),
		receiveQueue:   newReceiveQueue(),
		retransmission: newRetransmissionQueue(),
		sendQueue:      newSendQueue(connectionID),
		streams:        newStreamMap(),
		rtt:            internal.NewRTT(),
	}
	go c.pace()
	go c.tick()
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
	return c.conn.RemoteAddr()
}

func (c *connection) CloseWithError(code byte, message string) (err error) {
	_ = c.write(&frame.ConnectionClose{Code: code, Message: message})
	return c.internalClose(message)
}

func (c *connection) Context() context.Context {
	return c.ctx
}

func (c *connection) internalClose(message string) (err error) {
	c.once.Do(func() {
		for _, stream := range c.streams.all() {
			_ = stream.internalClose()
		}
		c.cancelFunc(errors.New(message))
		_ = c.conn.Close()
	})
	return
}

func (c *connection) createStream(streamID protocol.StreamID) (*Stream, error) {
	if c.streams.get(streamID) != nil {
		return nil, fmt.Errorf("stream %v already exists", streamID)
	}
	stream := newStream(c, streamID)
	c.streams.add(stream)
	return stream, nil
}

func (c *connection) pace() {
	defer c.CloseWithError(frame.ConnectionCloseInternal, "")
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.sendQueue.mu.Lock()
		if len(c.sendQueue.list) == 0 {
			c.sendQueue.cond.Wait()
		}

		c.sendQueue.mu.Unlock()
		if err := c.transmit(); err != nil {
			return
		}
	}
}

func (c *connection) tick() {
	ticker := time.NewTicker(time.Millisecond * 80)
	defer func() {
		ticker.Stop()
		_ = c.CloseWithError(frame.ConnectionCloseInternal, "")
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.acknowledge(); err != nil {
				return
			}

			if err := c.retransmit(); err != nil {
				return
			}

			if time.Since(time.Unix(0, c.activity.Load())) >= inactivityTimeout {
				_ = c.CloseWithError(frame.ConnectionCloseTimeout, "network inactivity")
				return
			}
		}
	}
}

func (c *connection) receive(sequenceID uint32, frames []frame.Frame) (err error) {
	if c.receiveQueue.exists(sequenceID) {
		c.ack.addDuplicate(sequenceID)
		return
	}

	for _, fr := range frames {
		if err := c.handler(fr); err != nil {
			return err
		}

		if err := c.handle(fr); err != nil {
			return err
		}
	}

	if sequenceID != 0 {
		c.ack.add(sequenceID)
		c.receiveQueue.store(sequenceID)
	}
	c.activity.Store(time.Now().UnixNano())
	//lint:ignore SA6002 ignore this for now.
	frame.Pool.Put(frames[:0])
	return
}

func (c *connection) handle(fr frame.Frame) (err error) {
	switch fr := fr.(type) {
	case *frame.Acknowledgement:
		var ackBytes uint64
		for i, r := range fr.Ranges {
			for j := r[0]; j <= r[1]; j++ {
				if entry := c.retransmission.remove(j); entry != nil {
					ackBytes += uint64(len(entry.payload))
					if i == len(fr.Ranges)-1 && j == r[1] {
						c.rtt.Add(time.Duration(time.Since(entry.timestamp).Nanoseconds()-fr.Delay) * time.Nanosecond)
					}
				}
			}
		}

		if ackBytes > 0 {
			c.cc.OnAck(ackBytes)
		}

		if fr.Type == frame.AcknowledgementWithGaps {
			for _, sequenceID := range frame.GenerateAcknowledgementGaps(fr.Ranges) {
				c.retransmission.nack(sequenceID)
			}
		}
	case *frame.ConnectionClose:
		if err := c.internalClose(fr.Message); err != nil {
			return err
		}
	case *frame.StreamData:
		if stream := c.streams.get(fr.StreamID); stream != nil {
			stream.receive(fr)
		}
	case *frame.StreamClose:
		if stream := c.streams.get(fr.StreamID); stream != nil {
			_ = stream.internalClose()
		}
	}
	frame.PutFrame(fr)
	return
}

func (c *connection) write(fr frame.Frame) (err error) {
	p, err := frame.PackSingle(fr)
	if err != nil {
		return err
	}
	c.sendQueue.add(p)
	return
}

func (c *connection) writeImmediately(fr frame.Frame) (err error) {
	pk, err := frame.PackSingle(fr)
	if err != nil {
		return err
	}

	if _, err := c.conn.Write(frame.Pack(c.connectionID, 0, 1, pk)); err != nil {
		return err
	}
	return
}

func (c *connection) acknowledge() (err error) {
	delay, list := c.ack.flush()
	if list == nil {
		return
	}
	ackType, ranges := frame.GenerateAcknowledgementRange(list)
	return c.writeImmediately(&frame.Acknowledgement{
		Type:   ackType,
		Delay:  delay,
		Ranges: ranges,
	})
}

func (c *connection) transmit() (err error) {
	sequenceID, pk := c.sendQueue.shift()
	if d := c.pacer.Delay(c.rtt.RTT(), uint64(len(pk)), c.cc.Cwnd()); d > 0 {
		time.Sleep(d)
	}

	if ch := c.cc.ScheduleSend(uint64(len(pk))); ch != nil {
		<-ch
	}

	c.cc.OnSend(uint64(len(pk)))
	c.pacer.OnSend(uint64(len(pk)))
	c.retransmission.add(sequenceID, pk)
	if _, err := c.conn.Write(pk); err != nil {
		return err
	}
	c.activity.Store(time.Now().UnixNano())
	return
}

func (c *connection) retransmit() (err error) {
	if pk := c.retransmission.shift(); pk != nil {
		c.cc.OnLoss()
		if _, err := c.conn.Write(pk); err != nil {
			return err
		}
	}
	return
}
