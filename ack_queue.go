package spectral

import (
	"slices"
	"sync"
	"time"
)

type ackQueue struct {
	sequenceID uint32
	sort       bool
	lastAck    time.Time
	list       []uint32
	mu         sync.Mutex
}

func newAckQueue() *ackQueue {
	return &ackQueue{}
}

func (a *ackQueue) add(sequenceID uint32) {
	a.mu.Lock()
	a.sequenceID++
	if !a.sort && sequenceID != a.sequenceID {
		a.sort = true
	}
	a.lastAck = time.Now()
	a.list = append(a.list, sequenceID)
	a.mu.Unlock()
}

func (a *ackQueue) addDuplicate(sequenceID uint32) {
	a.mu.Lock()
	a.sort = true
	a.list = append(a.list, sequenceID)
	a.mu.Unlock()
}

func (a *ackQueue) flush() (delay int64, list []uint32) {
	a.mu.Lock()
	if len(a.list) > 0 {
		delay = time.Since(a.lastAck).Nanoseconds()
		list = a.list
		if a.sort {
			slices.Sort(list)
		}
		a.sort = false
		a.lastAck = time.Time{}
		a.list = a.list[:0]
	}
	a.mu.Unlock()
	return
}
