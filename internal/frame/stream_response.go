package frame

import (
	"encoding/binary"
	"errors"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	StreamResponseSuccess = iota
	StreamResponseFailed
)

type StreamResponse struct {
	StreamID protocol.StreamID
	Response byte
}

func (fr *StreamResponse) ID() uint32 {
	return IDStreamResponse
}

func (fr *StreamResponse) Encode() ([]byte, error) {
	p := make([]byte, 9)
	binary.LittleEndian.PutUint64(p[0:8], uint64(fr.StreamID))
	p[8] = fr.Response
	return p, nil
}

func (fr *StreamResponse) Decode(p []byte) (int, error) {
	if len(p) < 9 {
		return 0, errors.New("not enough data to decode")
	}
	fr.StreamID = protocol.StreamID(binary.LittleEndian.Uint64(p))
	fr.Response = p[8]
	return 9, nil
}
