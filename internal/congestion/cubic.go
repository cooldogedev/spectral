//lint:file-ignore U1000 ignore all unused for now.

package congestion

import (
	"math"
	"time"

	"github.com/cooldogedev/spectral/internal/log"
)

const (
	cubicBeta = 0.7
	cubicC    = 0.4
)

func cubicK(wMax float64, mss uint64) float64 {
	return math.Cbrt(wMax / float64(mss) * (1.0 - cubicBeta) / cubicC)
}

func wCubic(t time.Duration, wMax float64, k float64, mss uint64) float64 {
	return cubicC * (math.Pow(t.Seconds()-k, 3) + wMax/float64(mss)) * float64(mss)
}

func wEst(t time.Duration, rtt time.Duration, wMax float64, mss uint64) float64 {
	return (wMax/float64(mss)*cubicBeta + 3.0*(1.0-cubicBeta)/(1.0+cubicBeta)*t.Seconds()/rtt.Seconds()) * float64(mss)
}

type cubic struct {
	window  uint64
	ssthres uint64
	mss     uint64
	cwndInc uint64
	wMax    float64
	k       float64
	logger  log.Logger
}

func newCubic(logger log.Logger, mss uint64) *cubic {
	window := initialWindow(mss)
	return &cubic{
		window:  window,
		ssthres: math.MaxUint64,
		mss:     mss,
		wMax:    float64(window),
		logger:  logger,
	}
}

func (c *cubic) OnAck(now, _, recoveryStartTime time.Time, rtt time.Duration, bytes uint64) {
	if c.window < c.ssthres {
		c.window += bytes
		c.logger.Log("congestion_window_increase", "cause", "slow_start", "window", c.window)
		return
	}

	t := now.Sub(recoveryStartTime)
	w := wCubic(t+rtt, c.wMax, c.k, c.mss)
	est := wEst(t, rtt, c.wMax, c.mss)
	cubicCwnd := c.window
	if w < est {
		cubicCwnd = max(cubicCwnd, uint64(est))
	} else if cubicCwnd < uint64(w) {
		cubicCwnd += uint64((w - float64(cubicCwnd)) / float64(cubicCwnd) * float64(c.mss))
	}

	c.cwndInc += cubicCwnd - c.window
	if c.cwndInc >= c.mss {
		c.window += c.mss
		c.cwndInc = 0
		c.logger.Log("congestion_window_increase", "cause", "congestion_avoidance", "window", c.window)
	}
}

func (c *cubic) OnCongestionEvent(_ time.Time, _ time.Time) {
	if float64(c.window) < c.wMax {
		c.wMax = float64(c.window) * (1.0 - cubicBeta) / 2.0
	} else {
		c.wMax = float64(c.window)
	}
	c.ssthres = max(uint64(c.wMax*cubicBeta), minimumWindow(c.mss))
	c.window = c.ssthres
	c.k = cubicK(c.wMax, c.mss)
	c.cwndInc = uint64(float64(c.cwndInc) * cubicBeta)
	c.logger.Log("congestion_window_decrease", "window", c.window)
}

func (c *cubic) SetMSS(mss uint64) {
	c.mss = mss
	c.window = max(c.window, minimumWindow(mss))
}

func (c *cubic) MSS() uint64 {
	return c.mss
}

func (c *cubic) Window() uint64 {
	return c.window
}
