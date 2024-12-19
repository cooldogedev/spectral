package spectral

import (
	"slices"
	"sync"
	"time"
)

const retransmissionAttempts = 3

type retransmissionEntry struct {
	sequenceID uint32
	payload    []byte
	sent       time.Time
	attempts   int
}

type retransmissionQueue struct {
	queue []*retransmissionEntry
	mu    sync.RWMutex
}

func newRetransmissionQueue() *retransmissionQueue {
	return &retransmissionQueue{}
}

func (r *retransmissionQueue) add(now time.Time, sequenceID uint32, p []byte) {
	r.mu.Lock()
	r.queue = append(r.queue, &retransmissionEntry{sequenceID: sequenceID, payload: p, sent: now})
	r.sort()
	r.mu.Unlock()
}

func (r *retransmissionQueue) remove(sequenceID uint32) *retransmissionEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := slices.IndexFunc(r.queue, func(e *retransmissionEntry) bool { return e.sequenceID == sequenceID })
	if index >= 0 {
		entry := r.queue[index]
		r.queue = slices.Delete(r.queue, index, index+1)
		return entry
	}
	return nil
}

func (r *retransmissionQueue) next(rto time.Duration) (t time.Time) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.queue) > 0 {
		return r.queue[0].sent.Add(rto)
	}
	return
}

func (r *retransmissionQueue) shift(now time.Time, rto time.Duration) (p []byte, t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queue) == 0 {
		return
	}

	entry := r.queue[0]
	if now.Sub(entry.sent) >= rto {
		sent := entry.sent
		entry.sent = now
		entry.attempts++
		if entry.attempts >= retransmissionAttempts {
			r.queue[0] = nil
			r.queue = r.queue[1:]
		} else {
			r.queue = append(r.queue[1:], entry)
			r.sort()
		}
		return entry.payload, sent
	}
	return
}

func (r *retransmissionQueue) clear() {
	r.mu.Lock()
	for i, entry := range r.queue {
		entry.payload = entry.payload[:0]
		entry.payload = nil
		r.queue[i] = nil
	}
	r.queue = r.queue[:0]
	r.queue = nil
	r.mu.Unlock()
}

func (r *retransmissionQueue) sort() {
	if len(r.queue) > 1 {
		slices.SortFunc(r.queue, func(a, b *retransmissionEntry) int {
			if a.sent.Before(b.sent) {
				return -1
			} else if b.sent.Before(a.sent) {
				return 1
			}
			return 0
		})
	}
}
