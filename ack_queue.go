package spectral

import (
	"slices"
	"time"

	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

type ackQueue struct {
	ranges  []frame.AcknowledgementRange
	max     uint32
	maxTime time.Time
	nextAck time.Time
}

func newAckQueue() *ackQueue {
	return &ackQueue{}
}

func (a *ackQueue) next() (t time.Time) {
	return a.nextAck
}

func (a *ackQueue) add(now time.Time, sequenceID uint32) {
	for i, r := range a.ranges {
		if sequenceID >= r[0] && sequenceID <= r[1] {
			return
		}

		if sequenceID == r[1]+1 {
			a.ranges[i][1] = sequenceID
			a.merge()
			return
		}

		if sequenceID+1 == r[0] {
			a.ranges[i][0] = sequenceID
			a.merge()
			return
		}
	}

	a.ranges = append(a.ranges, frame.AcknowledgementRange{sequenceID, sequenceID})
	if sequenceID > a.max {
		a.max = sequenceID
		a.maxTime = now
	}

	if a.nextAck.IsZero() {
		a.nextAck = now.Add(protocol.MaxAckDelay - protocol.TimerGranularity)
	}
}

func (a *ackQueue) merge() {
	if len(a.ranges) <= 1 {
		return
	}

	slices.SortFunc(a.ranges, func(a, b frame.AcknowledgementRange) int {
		if a[0] < b[0] {
			return -1
		} else if a[0] > b[0] {
			return 1
		}
		return 0
	})
	merged := make([]frame.AcknowledgementRange, 0, len(a.ranges))
	current := a.ranges[0]
	for _, next := range a.ranges[1:] {
		if next[0] <= current[1]+1 {
			current[1] = max(current[1], next[1])
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	a.ranges = append(merged, current)
}

func (a *ackQueue) flush(now time.Time, length int, append bool) (ranges []frame.AcknowledgementRange, maxSequenceID uint32, delay int64) {
	length = min(len(a.ranges), length)
	if length > 0 && (now.After(a.nextAck) || append) {
		ranges = a.ranges[:length]
		maxSequenceID = a.max
		delay = now.Sub(a.maxTime).Microseconds()
		if delay < 0 {
			delay = 0
		}

		a.ranges = a.ranges[length:]
		if len(a.ranges) == 0 {
			a.max = 0
			a.maxTime = time.Time{}
			a.nextAck = time.Time{}
		}
	}
	return
}
