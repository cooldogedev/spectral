package frame

type ConnectionRequest struct {
}

func (fr *ConnectionRequest) ID() uint32 {
	return IDConnectionRequest
}

func (fr *ConnectionRequest) Encode() (n []byte, err error) { return }

func (fr *ConnectionRequest) Decode(_ []byte) (n int, err error) { return }
