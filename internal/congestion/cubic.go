package congestion

import (
	"math"
	"sync"
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const (
	MaxCwnd = 1000000
	C       = 0.7
	Beta    = 0.4
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
		cwnd:     10 * protocol.MaxPacketSize,
		wMax:     10 * protocol.MaxPacketSize,
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
		c.cwnd = min(c.cwnd+bytes, MaxCwnd)
		return
	}

	if c.epochStart.IsZero() {
		c.epochStart = time.Now()
		c.k = math.Cbrt(c.wMax * (1.0 - Beta) / C)
	}

	elapsed := time.Since(c.epochStart).Seconds()
	cwnd := C*math.Pow(elapsed-c.k, 3) + c.wMax
	if cwnd > c.cwnd {
		c.cwnd = min(cwnd, MaxCwnd)
	}
}

func (c *Cubic) OnLoss(bytes float64) {
	c.mu.Lock()
	c.wMax = c.cwnd
	c.cwnd *= Beta
	c.inFlight = max(c.inFlight-bytes, 0)
	c.ssthresh = c.cwnd
	c.epochStart = time.Time{}
	c.k = math.Cbrt(c.wMax * (1.0 - Beta) / C)
	c.mu.Unlock()
}

func (c *Cubic) Cwnd() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cwnd - c.inFlight
}
