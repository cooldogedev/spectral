package frame

import (
	"encoding/binary"
	"errors"
)

type MTURequest struct {
	MTU uint64
}

func (fr *MTURequest) ID() uint32 {
	return IDMTURequest
}

func (fr *MTURequest) Encode() []byte {
	p := make([]byte, fr.MTU)
	binary.LittleEndian.PutUint64(p[0:8], fr.MTU)
	return p
}

func (fr *MTURequest) Decode(p []byte) (int, error) {
	if len(p) < 8 {
		return 0, errors.New("not enough data to decode")
	}
	fr.MTU = binary.LittleEndian.Uint64(p[0:8])
	return int(fr.MTU), nil
}

func (fr *MTURequest) Reset() {}
