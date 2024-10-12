package congestion

import (
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	minPacingDelay = time.Millisecond * 2
	minBurstSize   = 2
	maxBurstSize   = 10
)

type Pacer struct {
	capacity uint64
	tokens   uint64
	window   uint64
	rate     float64
	rtt      time.Duration
	prev     time.Time
}

func NewPacer() *Pacer {
	return &Pacer{prev: time.Now()}
}

func (p *Pacer) Delay(rtt time.Duration, bytes uint64, window uint64) time.Duration {
	if window != p.window || rtt != p.rtt {
		p.capacity = optimalCapacity(rtt, window)
		p.tokens = min(p.tokens, p.capacity)
		p.window = window
		p.rate = 1.25 * float64(window) / max(rtt.Seconds(), 1)
		p.rtt = rtt
	}

	if p.tokens >= bytes {
		return 0
	}

	now := time.Now()
	newTokens := p.rate * now.Sub(p.prev).Seconds()
	p.tokens = min(p.tokens+uint64(newTokens), p.capacity)
	p.prev = now
	if p.tokens >= bytes {
		return 0
	}
	delay := time.Duration(float64(bytes-p.tokens) / p.rate * float64(time.Second))
	return max(delay, minPacingDelay)
}

func (p *Pacer) OnSend(bytes uint64) {
	p.tokens = max(p.tokens-bytes, 0)
}

func optimalCapacity(rtt time.Duration, window uint64) uint64 {
	rttNs := max(rtt.Nanoseconds(), 1)
	capacity := (window * uint64(minPacingDelay.Nanoseconds())) / uint64(rttNs)
	return clamp(capacity, minBurstSize*protocol.MaxPacketSize, maxBurstSize*protocol.MaxPacketSize)
}

func clamp(value, min, max uint64) uint64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}
