package congestion

import (
	"math"
	"sync"
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	maxBurstPackets = 3

	initialWindow = protocol.MaxPacketSize * 32
	minWindow     = protocol.MaxPacketSize * 2
	maxWindow     = protocol.MaxPacketSize * 10000

	cubicBeta = 0.7
	cubicC    = 0.4
)

type Cubic struct {
	awaiting   uint64
	flight     uint64
	cwnd       uint64
	wMax       uint64
	ssthresh   uint64
	k          float64
	ch         chan struct{}
	epochStart time.Time
	mu         sync.RWMutex
}

func NewCubic() *Cubic {
	return &Cubic{
		cwnd:     initialWindow,
		wMax:     initialWindow,
		ssthresh: math.MaxUint64,
		ch:       make(chan struct{}, 1),
	}
}

func (c *Cubic) ScheduleSend(bytes uint64) chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cwnd-c.flight >= bytes {
		return nil
	}
	c.awaiting = bytes
	return c.ch
}

func (c *Cubic) OnSend(bytes uint64) {
	c.mu.Lock()
	c.awaiting = 0
	c.flight += bytes
	c.mu.Unlock()
}

func (c *Cubic) OnAck(bytes uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.flight = max(c.flight-bytes, 0)
	if c.awaiting != 0 && c.cwnd-c.flight >= c.awaiting {
		c.ch <- struct{}{}
	}

	if !c.shouldIncreaseWindow() {
		return
	}

	if c.ssthresh > c.cwnd {
		c.cwnd = min(c.cwnd+bytes, maxWindow)
		return
	}

	if c.epochStart.IsZero() {
		c.epochStart = time.Now()
		c.k = math.Cbrt(float64(c.wMax) * (1.0 - cubicBeta) / cubicC)
	}

	elapsed := time.Since(c.epochStart).Seconds()
	cwnd := uint64(cubicC*math.Pow(elapsed-c.k, 3) + float64(c.wMax))
	if cwnd > c.cwnd {
		c.cwnd = min(cwnd, maxWindow)
	}
}

func (c *Cubic) OnLoss() {
	c.mu.Lock()
	c.flight = 0
	if c.awaiting != 0 {
		c.ch <- struct{}{}
	}
	c.wMax = c.cwnd
	c.cwnd = max(uint64(float64(c.cwnd)*cubicBeta), minWindow)
	c.ssthresh = c.cwnd
	c.epochStart = time.Time{}
	c.k = math.Cbrt(float64(c.wMax) * (1.0 - cubicBeta) / cubicC)
	c.mu.Unlock()
}

func (c *Cubic) Cwnd() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cwnd
}

func (c *Cubic) shouldIncreaseWindow() bool {
	if c.flight >= c.cwnd {
		return true
	}
	availableBytes := c.cwnd - c.flight
	slowStartLimited := c.ssthresh > c.cwnd && c.flight > c.cwnd/2
	return slowStartLimited || availableBytes <= maxBurstPackets*protocol.MaxPacketSize
}
