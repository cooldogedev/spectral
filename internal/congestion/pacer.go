package congestion

import (
	"math"
	"time"
)

const (
	burstIntervalNanoseconds = 2_000_000
	minBurstSize             = 10
	maxBurstSize             = 256
)

type pacer struct {
	capacity uint64
	tokens   uint64
	mss      uint64
	window   uint64
	prev     time.Time
}

func newPacer(now time.Time) *pacer {
	return &pacer{prev: now}
}

func (p *pacer) timeUntilSend(now time.Time, rtt time.Duration, bytes uint64, mss uint64, window uint64) (t time.Time) {
	if mss != p.mss || window != p.window {
		p.capacity = optimalCapacity(rtt, mss, window)
		p.tokens = min(p.tokens, p.capacity)
		p.mss = mss
		p.window = window
	}

	if p.tokens >= bytes || window >= math.MaxUint32 {
		return
	}

	elapsed := now.Sub(p.prev)
	elapsedRTT := elapsed.Seconds() / rtt.Seconds()
	newTokens := float64(window) * 1.25 * elapsedRTT
	p.tokens = min(p.tokens+uint64(newTokens), p.capacity)
	p.prev = now
	if p.tokens >= bytes {
		return
	}
	unscaledDelay := uint64(rtt) * (min(bytes, p.capacity) - p.tokens) / window
	return p.prev.Add(time.Duration(unscaledDelay/5) * 4)
}

func (p *pacer) onSend(bytes uint64) {
	if p.tokens > bytes {
		p.tokens -= bytes
	} else {
		p.tokens = 0
	}
}

func optimalCapacity(rtt time.Duration, mss uint64, window uint64) uint64 {
	rttNs := max(rtt.Nanoseconds(), 1)
	capacity := (window * burstIntervalNanoseconds) / uint64(rttNs)
	return clamp(capacity, minBurstSize*mss, maxBurstSize*mss)
}
