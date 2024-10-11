package frame

import (
	"encoding/binary"
	"errors"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type StreamRequest struct {
	StreamID protocol.StreamID
}

func (fr *StreamRequest) ID() uint32 {
	return IDStreamRequest
}

func (fr *StreamRequest) Encode() ([]byte, error) {
	p := make([]byte, 8)
	binary.LittleEndian.PutUint64(p, uint64(fr.StreamID))
	return p, nil
}

func (fr *StreamRequest) Decode(p []byte) (int, error) {
	if len(p) < 8 {
		return 0, errors.New("not enough data to decode")
	}
	fr.StreamID = protocol.StreamID(binary.LittleEndian.Uint64(p))
	return 8, nil
}

func (fr *StreamRequest) Reset() {}
