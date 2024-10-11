package spectral

import (
	"sync"
	"time"
)

const (
	retransmissionAttempts = 3
	retransmissionDelay    = time.Second * 3
)

type retransmissionEntry struct {
	payload   []byte
	timestamp time.Time
	attempts  int
	nack      bool
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
	r.entries[sequenceID] = &retransmissionEntry{payload: p, timestamp: time.Now()}
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
		for sequenceID, entry := range r.entries {
			if entry.nack || now.Sub(entry.timestamp) >= retransmissionDelay {
				entry.timestamp = now
				entry.nack = false
				entry.attempts++
				if entry.attempts >= retransmissionAttempts {
					delete(r.entries, sequenceID)
				}
				p = entry.payload
				break
			}
		}
	}
	r.mu.Unlock()
	return
}
