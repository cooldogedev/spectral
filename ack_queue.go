package spectral

import (
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type ackQueue struct {
	queue   []uint32
	max     uint32
	maxTime time.Time
	nextAck time.Time
}

func newAckQueue() *ackQueue {
	return &ackQueue{}
}

func (a *ackQueue) add(now time.Time, sequenceID uint32) {
	a.queue = append(a.queue, sequenceID)
	if sequenceID > a.max {
		a.max = sequenceID
		a.maxTime = now
	}

	if a.nextAck.IsZero() {
		a.nextAck = now.Add(protocol.MaxAckDelay - protocol.TimerGranularity)
	}
}

func (a *ackQueue) next() (t time.Time) {
	return a.nextAck
}

func (a *ackQueue) flush(now time.Time) (list []uint32, maxSequenceID uint32, delay time.Duration) {
	if len(a.queue) > 0 && now.After(a.nextAck) {
		list = a.queue
		maxSequenceID = a.max
		delay = now.Sub(a.maxTime)
		if delay < 0 {
			delay = 0
		}
		a.queue = a.queue[:0]
		a.max = 0
		a.maxTime = time.Time{}
		a.nextAck = time.Time{}
	}
	return
}
