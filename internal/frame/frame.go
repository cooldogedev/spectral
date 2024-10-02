package frame

type Frame interface {
	ID() uint32
	Encode() ([]byte, error)
	Decode(p []byte) (int, error)
}

type ResettableFrame interface {
	Frame
	Reset()
}
