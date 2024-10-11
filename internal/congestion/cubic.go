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
	inFlight   uint64
	cwnd       uint64
	wMax       uint64
	ssthresh   uint64
	k          float64
	epochStart time.Time
	mu         sync.RWMutex
}

func NewCubic() *Cubic {
	return &Cubic{
		cwnd:     initialWindow,
		wMax:     initialWindow,
		ssthresh: math.MaxUint64,
	}
}

func (c *Cubic) CanSend(bytes uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cwnd-c.inFlight >= bytes
}

func (c *Cubic) OnSend(bytes uint64) {
	c.mu.Lock()
	c.inFlight += bytes
	c.mu.Unlock()
}

func (c *Cubic) OnAck(bytes uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inFlight = max(c.inFlight-bytes, 0)
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
	c.inFlight = 0
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
	if c.inFlight >= c.cwnd {
		return true
	}
	availableBytes := c.cwnd - c.inFlight
	slowStartLimited := c.ssthresh > c.cwnd && c.inFlight > c.cwnd/2
	return slowStartLimited || availableBytes <= maxBurstPackets*protocol.MaxPacketSize
}
