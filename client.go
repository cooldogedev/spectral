package spectral

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/log"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type ClientConnection struct {
	*connection
	response        chan *frame.ConnectionResponse
	streamResponses map[protocol.StreamID]chan *frame.StreamResponse
	streamID        protocol.StreamID
	mu              sync.RWMutex
}

func newClientConnection(conn *udpConn, peerAddr *net.UDPAddr, ctx context.Context) *ClientConnection {
	c := &ClientConnection{
		connection:      newConnection(conn, peerAddr, -1, ctx, log.PerspectiveClient),
		response:        make(chan *frame.ConnectionResponse, 1),
		streamResponses: make(map[protocol.StreamID]chan *frame.StreamResponse),
	}
	c.connection.handler = c.handle
	return c
}

func (c *ClientConnection) OpenStream(ctx context.Context) (*Stream, error) {
	ch := make(chan *frame.StreamResponse, 1)
	c.mu.Lock()
	streamID := c.streamID
	c.streamID++
	c.streamResponses[streamID] = ch
	c.mu.Unlock()
	defer func() {
		close(ch)
		c.mu.Lock()
		delete(c.streamResponses, streamID)
		c.mu.Unlock()
	}()

	c.logger.Log("stream_open_request", "streamID", streamID)
	if err := c.writeControl(&frame.StreamRequest{StreamID: streamID}, true); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case response := <-ch:
		if response.Response == frame.StreamResponseFailed {
			c.logger.Log("stream_open_fail", "streamID", streamID)
			return nil, errors.New("failed to open stream")
		}

		stream, err := c.createStream(streamID)
		if err != nil {
			return nil, err
		}
		c.logger.Log("stream_open_success", "streamID", streamID)
		return stream, nil
	}
}

func (c *ClientConnection) handle(fr frame.Frame) (err error) {
	switch fr := fr.(type) {
	case *frame.ConnectionResponse:
		c.response <- fr
	case *frame.StreamResponse:
		c.mu.RLock()
		ch, ok := c.streamResponses[fr.StreamID]
		c.mu.RUnlock()
		if ok {
			select {
			case ch <- fr:
			default:
			}
		} else {
			c.logger.Log("stream_response_unknown", "streamID", fr.StreamID)
		}
	}
	return
}
