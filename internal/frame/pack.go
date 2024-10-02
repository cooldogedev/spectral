package frame

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cooldogedev/spectral/internal/protocol"
)

func PackSingle(fr Frame) ([]byte, error) {
	p, err := fr.Encode()
	if err != nil {
		return nil, err
	}
	id := make([]byte, 4)
	binary.LittleEndian.PutUint32(id, fr.ID())
	return append(id, p...), nil
}

func Pack(connectionID protocol.ConnectionID, sequenceID uint32, total uint32, frames []byte) []byte {
	header := make([]byte, protocol.PacketHeaderSize)
	copy(header, protocol.Magic)
	binary.LittleEndian.PutUint64(header[4:12], uint64(connectionID))
	binary.LittleEndian.PutUint32(header[12:16], sequenceID)
	binary.LittleEndian.PutUint32(header[16:20], total)
	return append(header, frames...)
}

func Unpack(p []byte) (connectionID protocol.ConnectionID, sequenceID uint32, frames []Frame, err error) {
	if len(p) < protocol.PacketHeaderSize || string(p[0:4]) != string(protocol.Magic) {
		return 0, 0, nil, errors.New("invalid header")
	}

	var totalFrames uint32
	var frameID uint32
	connectionID = protocol.ConnectionID(binary.LittleEndian.Uint64(p[4:12]))
	sequenceID = binary.LittleEndian.Uint32(p[12:16])
	totalFrames = binary.LittleEndian.Uint32(p[16:20])
	frames = Pool.Get().([]Frame)[:totalFrames]
	offset := 20
	for i := 0; i < int(totalFrames); i++ {
		frameID = binary.LittleEndian.Uint32(p[offset : offset+4])
		offset += 4
		frames[i], err = GetFrame(frameID)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("unknown frame %v", frameID)
		}

		n, err := frames[i].Decode(p[offset:])
		if err != nil {
			return 0, 0, nil, fmt.Errorf("error while decoding frame %v: %v", frameID, err)
		}
		offset += n
	}
	return
}
