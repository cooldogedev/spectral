package congestion

import (
	"math"
	"sync"
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	initialWindow = protocol.MaxPacketSize * 32
	minWindow     = protocol.MaxPacketSize * 2
	maxWindow     = protocol.MaxPacketSize * 10000

	cubicC    = 0.7
	cubicBeta = 0.4
)

type Cubic struct {
	inFlight   float64
	cwnd       float64
	wMax       float64
	ssthresh   float64
	k          float64
	epochStart time.Time
	mu         sync.RWMutex
}

func NewCubic() *Cubic {
	return &Cubic{
		cwnd:     initialWindow,
		wMax:     initialWindow,
		ssthresh: math.Inf(1),
	}
}

func (c *Cubic) OnSend(bytes float64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cwnd-c.inFlight >= bytes {
		c.inFlight += bytes
		return true
	}
	return false
}

func (c *Cubic) OnAck(bytes float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inFlight = max(c.inFlight-bytes, 0)
	if c.ssthresh > c.cwnd {
		c.cwnd = min(c.cwnd+bytes, maxWindow)
		return
	}

	if c.epochStart.IsZero() {
		c.epochStart = time.Now()
		c.k = math.Cbrt(c.wMax * (1.0 - cubicBeta) / cubicC)
	}

	elapsed := time.Since(c.epochStart).Seconds()
	cwnd := cubicC*math.Pow(elapsed-c.k, 3) + c.wMax
	if cwnd > c.cwnd {
		c.cwnd = min(cwnd, maxWindow)
	}
}

func (c *Cubic) OnLoss(bytes float64) {
	c.mu.Lock()
	c.wMax = c.cwnd
	c.cwnd = max(c.cwnd*cubicBeta, minWindow)
	c.inFlight = max(c.inFlight-bytes, 0)
	c.ssthresh = c.cwnd
	c.epochStart = time.Time{}
	c.k = math.Cbrt(c.wMax * (1.0 - cubicBeta) / cubicC)
	c.mu.Unlock()
}

func (c *Cubic) Cwnd() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cwnd
}

func (c *Cubic) InFlight() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inFlight
}
