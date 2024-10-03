package congestion

import "time"

const bytesPerToken = 512

type Pacer struct {
	capacity   int
	tokens     int
	interval   time.Duration
	lastRefill time.Time
}

func NewPacer(interval time.Duration, capacity int) *Pacer {
	return &Pacer{
		capacity:   capacity,
		tokens:     capacity,
		interval:   interval,
		lastRefill: time.Now(),
	}
}

func (p *Pacer) Consume(bytes int) time.Duration {
	now := time.Now()
	elapsed := now.Sub(p.lastRefill)
	if elapsed >= p.interval {
		p.tokens = min(p.tokens+int(elapsed/p.interval), p.capacity)
		p.lastRefill = now
	}

	tokensNeeded := (bytes + bytesPerToken - 1) / bytesPerToken
	if p.tokens >= tokensNeeded {
		p.tokens -= tokensNeeded
		return 0
	}
	return time.Duration(tokensNeeded-p.tokens) * p.interval
}

func (p *Pacer) SetInterval(interval time.Duration) {
	if interval > 0 {
		p.interval = interval
	}
}
