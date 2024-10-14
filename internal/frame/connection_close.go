package frame

import (
	"encoding/binary"
	"errors"
)

const (
	ConnectionCloseApplication = iota
	ConnectionCloseGraceful
	ConnectionCloseTimeout
	ConnectionCloseInternal
)

type ConnectionClose struct {
	Code    byte
	Message string
}

func (fr *ConnectionClose) ID() uint32 {
	return IDConnectionClose
}

func (fr *ConnectionClose) Encode() []byte {
	messageLength := uint32(len(fr.Message))
	p := make([]byte, 1+4+messageLength)
	p[0] = fr.Code
	binary.LittleEndian.PutUint32(p[1:5], messageLength)
	copy(p[5:], fr.Message)
	return p
}

func (fr *ConnectionClose) Decode(p []byte) (int, error) {
	if len(p) < 5 {
		return 0, errors.New("not enough data to decode")
	}

	fr.Code = p[0]
	messageLength := binary.LittleEndian.Uint32(p[1:5])
	if len(p) < int(5+messageLength) {
		return 0, errors.New("not enough data to decode message")
	}
	fr.Message = string(p[5 : 5+messageLength])
	return 5 + int(messageLength), nil
}

func (fr *ConnectionClose) Reset() {}
