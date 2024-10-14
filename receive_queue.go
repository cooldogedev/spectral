package spectral

type receiveQueue struct {
	expected uint32
	queue    map[uint32]bool
}

func newReceiveQueue() *receiveQueue {
	return &receiveQueue{
		expected: 1,
		queue:    make(map[uint32]bool),
	}
}

func (r *receiveQueue) add(sequenceID uint32) bool {
	if r.exists(sequenceID) {
		return false
	}
	r.queue[sequenceID] = true
	r.merge()
	return true
}

func (r *receiveQueue) exists(sequenceID uint32) bool {
	if r.expected > sequenceID {
		return true
	}
	_, ok := r.queue[sequenceID]
	return ok
}

func (r *receiveQueue) merge() {
	for {
		if _, ok := r.queue[r.expected]; !ok {
			break
		}
		delete(r.queue, r.expected)
		r.expected++
	}
}
