package spectral

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/cooldogedev/spectral/internal/frame"
)

func Dial(ctx context.Context, address string) (Connection, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	uConn, err := newUDPConn(conn, false)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	c := newClientConnection(uConn, addr, context.Background())
	c.logger.Log("connection_request", "addr", address)
	if err := c.writeControl(&frame.ConnectionRequest{}, true); err != nil {
		_ = c.CloseWithError(frame.ConnectionCloseInternal, "failed to send connection request")
		return nil, err
	}

	go uConn.Read(func(dgram *datagram) (err error) {
		defer dgram.reset()
		_, sequenceID, frames, err := frame.Unpack(dgram.b)
		if err != nil {
			c.logger.Log("unpack_err", "err", err.Error())
			return err
		}

		select {
		case <-c.ctx.Done():
			return context.Cause(c.ctx)
		default:
			c.packets <- &receivedPacket{sequenceID, frames, time.Now()}
			return
		}
	})
	select {
	case <-ctx.Done():
		c.logger.Log("connection_request_timeout")
		_ = c.CloseWithError(frame.ConnectionCloseInternal, fmt.Sprintf("dialer context: %v", context.Cause(ctx).Error()))
		return nil, context.Cause(ctx)
	case response := <-c.response:
		if response.Response == frame.ConnectionResponseFailed {
			c.logger.Log("connection_request_fail")
			_ = c.CloseWithError(frame.ConnectionCloseInternal, "failed to open connection")
			return nil, errors.New("failed to open connection")
		}
		c.connectionID.Store(int64(response.ConnectionID))
		c.logger.SetConnectionID(response.ConnectionID)
		c.logger.Log("connection_request_success")
		return c, nil
	}
}
