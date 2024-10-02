package spectral

const maxSequenceID = 1 << 32

type receiveQueue [maxSequenceID / 8]byte

func newReceiveQueue() *receiveQueue {
	return &receiveQueue{}
}

func (r *receiveQueue) store(sequenceID uint32) {
	r[sequenceID/8] |= 1 << (sequenceID % 8)
}

func (r *receiveQueue) exists(sequenceID uint32) bool {
	return r[sequenceID/8]&(1<<(sequenceID%8)) != 0
}
