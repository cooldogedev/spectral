package spectral

import (
	"sync"
	"time"
)

const retransmissionDelay = time.Second * 5

type retransmissionEntry struct {
	p    []byte
	nack bool
	t    time.Time
}

type retransmissionQueue struct {
	entries map[uint32]*retransmissionEntry
	mu      sync.Mutex
}

func newRetransmissionQueue() *retransmissionQueue {
	return &retransmissionQueue{
		entries: make(map[uint32]*retransmissionEntry),
	}
}

func (r *retransmissionQueue) add(sequenceID uint32, p []byte) {
	r.mu.Lock()
	r.entries[sequenceID] = &retransmissionEntry{p: p, t: time.Now()}
	r.mu.Unlock()
}

func (r *retransmissionQueue) nack(sequenceID uint32) {
	r.mu.Lock()
	if entry, ok := r.entries[sequenceID]; ok {
		entry.nack = true
	}
	r.mu.Unlock()
}

func (r *retransmissionQueue) remove(sequenceID uint32) (e *retransmissionEntry) {
	r.mu.Lock()
	entry, ok := r.entries[sequenceID]
	if ok {
		e = entry
		delete(r.entries, sequenceID)
	}
	r.mu.Unlock()
	return
}

func (r *retransmissionQueue) shift() (p []byte) {
	r.mu.Lock()
	if len(r.entries) > 0 {
		now := time.Now()
		for _, entry := range r.entries {
			if entry.nack || now.Sub(entry.t) >= retransmissionDelay {
				entry.nack = false
				entry.t = now
				p = entry.p
			}
		}
	}
	r.mu.Unlock()
	return
}
