package frame

import (
	"encoding/binary"
	"errors"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type StreamData struct {
	StreamID   protocol.StreamID
	SequenceID uint32
	Total      uint32
	Offset     uint32
	Payload    []byte
}

func (fr *StreamData) ID() uint32 {
	return IDStreamData
}

func (fr *StreamData) Encode() ([]byte, error) {
	payloadLength := uint32(len(fr.Payload))
	p := make([]byte, 8+4+4+4+4+payloadLength)
	binary.LittleEndian.PutUint64(p[0:8], uint64(fr.StreamID))
	binary.LittleEndian.PutUint32(p[8:12], fr.SequenceID)
	binary.LittleEndian.PutUint32(p[12:16], fr.Total)
	binary.LittleEndian.PutUint32(p[16:20], fr.Offset)
	binary.LittleEndian.PutUint32(p[20:24], payloadLength)
	copy(p[24:], fr.Payload)
	return p, nil
}

func (fr *StreamData) Decode(p []byte) (int, error) {
	if len(p) < 24 {
		return 0, errors.New("not enough data to decode")
	}

	fr.StreamID = protocol.StreamID(binary.LittleEndian.Uint64(p[0:8]))
	fr.SequenceID = binary.LittleEndian.Uint32(p[8:12])
	fr.Total = binary.LittleEndian.Uint32(p[12:16])
	fr.Offset = binary.LittleEndian.Uint32(p[16:20])
	payloadLength := binary.LittleEndian.Uint32(p[20:24])
	if len(p) < int(24+payloadLength) {
		return 24, errors.New("not enough data to decode payload")
	}
	fr.Payload = append(fr.Payload, p[24:24+payloadLength]...)
	return 24 + int(payloadLength), nil
}

func (fr *StreamData) Reset() {
	fr.StreamID = 0
	fr.SequenceID = 0
	fr.Total = 0
	fr.Offset = 0
	fr.Payload = fr.Payload[:0]
}
