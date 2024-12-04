package spectral

import (
	"context"
	"errors"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type Listener struct {
	conn                *udpConn
	connections         map[protocol.ConnectionID]*ServerConnection
	connectionsMu       sync.Mutex
	connectionID        protocol.ConnectionID
	incomingConnections chan *ServerConnection
	ctx                 context.Context
	cancelFunc          context.CancelFunc
	once                sync.Once
}

func newListener(conn *udpConn) *Listener {
	listener := &Listener{
		conn:                conn,
		connections:         make(map[protocol.ConnectionID]*ServerConnection),
		incomingConnections: make(chan *ServerConnection, 100),
	}
	listener.ctx, listener.cancelFunc = context.WithCancel(context.Background())
	go conn.Read(func(dgram *datagram) (err error) {
		defer dgram.reset()
		connectionID, sequenceID, frames, err := frame.Unpack(dgram.b)
		if err != nil {
			return nil
		}

		listener.connectionsMu.Lock()
		defer listener.connectionsMu.Unlock()
		c, ok := listener.connections[connectionID]
		if !ok && slices.ContainsFunc(frames, func(fr frame.Frame) bool { return fr.ID() == frame.IDConnectionRequest }) {
			c = newServerConnection(conn, dgram.peerAddr, listener.connectionID, listener.ctx)
			c.logger.Log("connection_accepted", "addr", dgram.peerAddr.String())
			listener.connections[listener.connectionID] = c
			listener.connectionID++
			listener.incomingConnections <- c
			go func() {
				<-c.ctx.Done()
				listener.connectionsMu.Lock()
				delete(listener.connections, protocol.ConnectionID(c.connectionID.Load()))
				listener.connectionsMu.Unlock()
			}()
		}

		if c == nil {
			return
		}

		select {
		case <-listener.ctx.Done():
			return context.Cause(listener.ctx)
		case <-c.ctx.Done():
		default:
			c.packets <- &receivedPacket{sequenceID, frames, time.Now()}
		}
		return
	})
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

	c, err := newUDPConn(conn, true)
	if err != nil {
		return nil, err
	}
	return newListener(c), nil
}

func (l *Listener) Accept(ctx context.Context) (Connection, error) {
	select {
	case <-l.ctx.Done():
		return nil, errors.New("listener closed")
	case <-ctx.Done():
		return nil, context.Cause(ctx)
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
