package spectral

import (
	"sync"
	"time"
)

type ackQueue struct {
	lastAck time.Time
	list    []uint32
	mu      sync.Mutex
}

func newAckQueue() *ackQueue {
	return &ackQueue{}
}

func (a *ackQueue) add(sequenceID uint32) {
	a.mu.Lock()
	a.lastAck = time.Now()
	a.list = append(a.list, sequenceID)
	a.mu.Unlock()
}

func (a *ackQueue) flush() (delay int64, list []uint32) {
	a.mu.Lock()
	if len(a.list) >= 0 {
		delay = time.Since(a.lastAck).Nanoseconds()
		list = a.list
		a.list = nil
	}
	a.mu.Unlock()
	return
}
