package frame

type Frame interface {
	ID() uint32
	Encode() []byte
	Decode(p []byte) (int, error)
	Reset()
}
