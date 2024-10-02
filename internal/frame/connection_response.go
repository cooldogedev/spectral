package frame

import (
	"encoding/binary"
	"errors"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	ConnectionResponseSuccess = iota
	ConnectionResponseFailed
)

type ConnectionResponse struct {
	ConnectionID protocol.ConnectionID
	Response     byte
}

func (fr *ConnectionResponse) ID() uint32 {
	return IDConnectionResponse
}

func (fr *ConnectionResponse) Encode() ([]byte, error) {
	p := make([]byte, 9)
	binary.LittleEndian.PutUint64(p[0:8], uint64(fr.ConnectionID))
	p[8] = fr.Response
	return p, nil
}

func (fr *ConnectionResponse) Decode(p []byte) (int, error) {
	if len(p) < 9 {
		return 0, errors.New("not enough data to decode")
	}
	fr.ConnectionID = protocol.ConnectionID(binary.LittleEndian.Uint64(p[0:8]))
	fr.Response = p[8]
	return 9, nil
}
