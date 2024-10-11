package spectral

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cooldogedev/spectral/internal"
	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type ClientConnection struct {
	*connection
	response        chan *frame.ConnectionResponse
	streamResponses map[protocol.StreamID]chan *frame.StreamResponse
	streamID        protocol.StreamID
	mu              sync.Mutex
}

func newClientConnection(conn *internal.Conn, ctx context.Context) *ClientConnection {
	c := &ClientConnection{
		connection:      newConnection(conn, -1, ctx),
		response:        make(chan *frame.ConnectionResponse),
		streamResponses: make(map[protocol.StreamID]chan *frame.StreamResponse),
	}
	c.connection.handler = c.handle
	return c
}

func (c *ClientConnection) OpenStream(ctx context.Context) (*Stream, error) {
	c.mu.Lock()
	streamID := c.streamID
	c.streamID++
	ch := make(chan *frame.StreamResponse, 1)
	c.streamResponses[streamID] = ch
	c.mu.Unlock()
	if err := c.write(&frame.StreamRequest{StreamID: streamID}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-ch:
		if response.Response == frame.StreamResponseFailed {
			return nil, errors.New("failed to open stream")
		}
		return c.createStream(streamID)
	}
}

func (c *ClientConnection) handle(fr frame.Frame) (err error) {
	switch fr := fr.(type) {
	case *frame.ConnectionResponse:
		c.response <- fr
	case *frame.StreamResponse:
		c.mu.Lock()
		ch, ok := c.streamResponses[fr.StreamID]
		delete(c.streamResponses, fr.StreamID)
		c.mu.Unlock()
		if !ok {
			return fmt.Errorf("received an unknown stream response for %v", fr.StreamID)
		}
		ch <- fr
	}
	return
}
