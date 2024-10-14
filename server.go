package spectral

import (
	"context"
	"net"

	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/log"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type ServerConnection struct {
	*connection
	streamRequests chan *frame.StreamRequest
}

func newServerConnection(conn *udpConn, peerAddr *net.UDPAddr, connectionID protocol.ConnectionID, ctx context.Context) *ServerConnection {
	c := &ServerConnection{
		connection:     newConnection(conn, peerAddr, connectionID, ctx, log.PerspectiveServer),
		streamRequests: make(chan *frame.StreamRequest, 100),
	}
	c.connection.handler = c.handle
	c.logger.SetConnectionID(connectionID)
	return c
}

func (c *ServerConnection) AcceptStream(ctx context.Context) (*Stream, error) {
	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case request := <-c.streamRequests:
		c.logger.Log("stream_accept", "streamID", request.StreamID)
		stream, err := c.createStream(request.StreamID)
		if err != nil {
			return nil, err
		}

		if err := c.writeControl(&frame.StreamResponse{StreamID: request.StreamID, Response: frame.StreamResponseSuccess}, true); err != nil {
			return nil, err
		}
		c.logger.Log("stream_accept_success", "streamID", request.StreamID)
		return stream, nil
	}
}

func (c *ServerConnection) handle(fr frame.Frame) (err error) {
	switch fr := fr.(type) {
	case *frame.ConnectionRequest:
		if err := c.writeControl(&frame.ConnectionResponse{ConnectionID: protocol.ConnectionID(c.connectionID.Load()), Response: frame.ConnectionResponseSuccess}, true); err != nil {
			return err
		}
	case *frame.StreamRequest:
		c.streamRequests <- fr
	}
	return
}
