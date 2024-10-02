package spectral

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/cooldogedev/spectral/internal"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
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

	if err := conn.SetReadBuffer(protocol.ReceiveBufferSize); err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := conn.SetWriteBuffer(protocol.SendBufferSize); err != nil {
		_ = conn.Close()
		return nil, err
	}

	c := newClientConnection(internal.NewConn(conn, addr, true), context.Background())
	if err := c.write(&frame.ConnectionRequest{}); err != nil {
		_ = c.CloseWithError(frame.ConnectionCloseInternal, "failed to send connection request")
		return nil, err
	}

	go func() {
		defer c.CloseWithError(frame.ConnectionCloseInternal, "error while reading")
		p := make([]byte, 1500)
		for {
			n, _, err := conn.ReadFromUDP(p)
			if err != nil {
				return
			}

			_, sequenceID, frames, err := frame.Unpack(p[:n])
			if err != nil {
				return
			}

			if err := c.receive(sequenceID, frames); err != nil {
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		_ = c.CloseWithError(frame.ConnectionCloseInternal, fmt.Sprintf("dialer context: %v", ctx.Err().Error()))
		return nil, ctx.Err()
	case response := <-c.response:
		if response.Response == frame.ConnectionResponseFailed {
			_ = c.CloseWithError(frame.ConnectionCloseInternal, "failed to open connection")
			return nil, errors.New("failed to open connection")
		}
		c.connectionID = response.ConnectionID
		c.sendQueue.connectionID = response.ConnectionID
		return c, nil
	}
}
