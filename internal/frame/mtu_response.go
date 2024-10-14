package frame

import (
	"encoding/binary"
	"errors"
)

type MTUResponse struct {
	MTU uint64
}

func (fr *MTUResponse) ID() uint32 {
	return IDMTUResponse
}

func (fr *MTUResponse) Encode() []byte {
	p := make([]byte, 8)
	binary.LittleEndian.PutUint64(p[0:8], fr.MTU)
	return p
}

func (fr *MTUResponse) Decode(p []byte) (int, error) {
	if len(p) < 8 {
		return 0, errors.New("not enough data to decode")
	}
	fr.MTU = binary.LittleEndian.Uint64(p[0:8])
	return 8, nil
}

func (fr *MTUResponse) Reset() {}
