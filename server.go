package spectral

import (
	"context"

	"github.com/cooldogedev/spectral/internal"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type ServerConnection struct {
	*connection
	streamRequests chan *frame.StreamRequest
}

func newServerConnection(conn *internal.Conn, connectionID protocol.ConnectionID, ctx context.Context) *ServerConnection {
	return &ServerConnection{
		connection:     newConnection(conn, connectionID, ctx),
		streamRequests: make(chan *frame.StreamRequest, 100),
	}
}

func (c *ServerConnection) AcceptStream(ctx context.Context) (*Stream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case request := <-c.streamRequests:
		stream, err := c.createStream(request.StreamID)
		if err != nil {
			return nil, err
		}

		if err := c.write(&frame.StreamResponse{StreamID: request.StreamID, Response: frame.StreamResponseSuccess}); err != nil {
			return nil, err
		}
		return stream, nil
	}
}

func (c *ServerConnection) receive(sequenceID uint32, frames []frame.Frame) (err error) {
	if c.receivedQueue.exists(sequenceID) {
		return
	}

	for _, fr := range frames {
		if err := c.handle(fr); err != nil {
			return err
		}
	}
	return c.connection.receive(sequenceID, frames)
}

func (c *ServerConnection) handle(fr frame.Frame) (err error) {
	switch fr := fr.(type) {
	case *frame.ConnectionRequest:
		if err := c.write(&frame.ConnectionResponse{ConnectionID: c.connectionID, Response: frame.ConnectionResponseSuccess}); err != nil {
			return err
		}
	case *frame.StreamRequest:
		c.streamRequests <- fr
	}
	return
}
