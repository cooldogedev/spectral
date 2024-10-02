package spectral

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"

	"github.com/cooldogedev/spectral/internal"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type Listener struct {
	conn                *net.UDPConn
	connections         map[protocol.ConnectionID]*ServerConnection
	connectionsMu       sync.Mutex
	connectionID        protocol.ConnectionID
	incomingConnections chan *ServerConnection
	ctx                 context.Context
	cancelFunc          context.CancelFunc
	once                sync.Once
}

func newListener(conn *net.UDPConn) *Listener {
	listener := &Listener{
		conn:                conn,
		connections:         make(map[protocol.ConnectionID]*ServerConnection),
		incomingConnections: make(chan *ServerConnection, 100),
	}
	listener.ctx, listener.cancelFunc = context.WithCancel(context.Background())
	go func() {
		defer listener.Close()
		p := make([]byte, 1500)
		for {
			n, addr, err := conn.ReadFromUDP(p)
			if err != nil {
				return
			}

			connectionID, sequenceID, frames, err := frame.Unpack(p[:n])
			if err != nil {
				continue
			}

			listener.connectionsMu.Lock()
			c, ok := listener.connections[connectionID]
			if !ok && slices.ContainsFunc(frames, func(fr frame.Frame) bool { return fr.ID() == frame.IDConnectionRequest }) {
				c = newServerConnection(internal.NewConn(conn, addr, false), listener.connectionID, listener.ctx)
				connectionID = listener.connectionID
				listener.connections[listener.connectionID] = c
				listener.connectionID++
				listener.incomingConnections <- c
				go func() {
					<-c.ctx.Done()
					listener.connectionsMu.Lock()
					delete(listener.connections, c.connectionID)
					listener.connectionsMu.Unlock()
				}()
			}

			listener.connectionsMu.Unlock()
			if c == nil {
				continue
			}

			if err := c.receive(sequenceID, frames); err != nil {
				_ = c.CloseWithError(frame.ConnectionCloseInternal, fmt.Sprintf("error while handling frame: %v", err.Error()))
				continue
			}
		}
	}()
	return listener
}

func Listen(address string) (*Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
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
	return newListener(conn), nil
}

func (l *Listener) Accept(ctx context.Context) (Connection, error) {
	select {
	case <-l.ctx.Done():
		return nil, errors.New("listener closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-l.incomingConnections:
		return conn, nil
	}
}

func (l *Listener) Close() (err error) {
	l.once.Do(func() {
		for i, conn := range l.connections {
			_ = conn.CloseWithError(frame.ConnectionCloseGraceful, "closed listener")
			delete(l.connections, i)
		}
		l.cancelFunc()
		_ = l.conn.Close()
	})
	return
}
