package frame

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cooldogedev/spectral/internal/protocol"
)

func PackSingle(fr Frame) []byte {
	id := make([]byte, 4)
	binary.LittleEndian.PutUint32(id, fr.ID())
	return append(id, fr.Encode()...)
}

func Pack(connectionID protocol.ConnectionID, sequenceID uint32, frames []byte) []byte {
	header := make([]byte, protocol.PacketHeaderSize)
	copy(header, protocol.Magic)
	binary.LittleEndian.PutUint64(header[4:12], uint64(connectionID))
	binary.LittleEndian.PutUint32(header[12:16], sequenceID)
	return append(header, frames...)
}

func Unpack(p []byte) (connectionID protocol.ConnectionID, sequenceID uint32, frames []Frame, err error) {
	length := len(p)
	if length < protocol.PacketHeaderSize || string(p[0:4]) != string(protocol.Magic) {
		return 0, 0, nil, errors.New("invalid header")
	}

	var frameID uint32
	connectionID = protocol.ConnectionID(binary.LittleEndian.Uint64(p[4:12]))
	sequenceID = binary.LittleEndian.Uint32(p[12:16])
	offset := 16
	for length > offset {
		frameID = binary.LittleEndian.Uint32(p[offset : offset+4])
		offset += 4
		fr, err := getFrame(frameID)
		if err != nil {
			return 0, 0, nil, err
		}

		n, err := fr.Decode(p[offset:])
		if err != nil {
			return 0, 0, nil, fmt.Errorf("error while decoding frame %v: %v", frameID, err)
		}
		frames = append(frames, fr)
		offset += n
	}
	return
}
