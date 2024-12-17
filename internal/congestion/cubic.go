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
	cwnd           uint64
	ssthres        uint64
	maxSegmentSize uint64
	cwndInc        uint64
	wMax           float64
	k              float64
	logger         log.Logger
}

func newCubic(logger log.Logger, mss uint64) *cubic {
	window := initialWindow(mss)
	return &cubic{
		cwnd:           window,
		ssthres:        math.MaxUint64,
		maxSegmentSize: mss,
		wMax:           float64(window),
		logger:         logger,
	}
}

func (c *cubic) onAck(now, _, recoveryStartTime time.Time, rtt *RTT, bytes, flight uint64) {
	if !shouldIncreaseWindow(flight, c.cwnd, c.ssthres) {
		return
	}

	if c.cwnd < c.ssthres {
		c.cwnd += bytes
		c.logger.Log("congestion_window_increase", "cause", "slow_start", "window", c.cwnd)
		return
	}

	smoothedRTT := rtt.SRTT()
	t := now.Sub(recoveryStartTime)
	w := wCubic(t+smoothedRTT, c.wMax, c.k, c.maxSegmentSize)
	est := wEst(t, smoothedRTT, c.wMax, c.maxSegmentSize)
	cubicCwnd := c.cwnd
	if w < est {
		cubicCwnd = max(cubicCwnd, uint64(est))
	} else if cubicCwnd < uint64(w) {
		cubicCwnd += uint64((w - float64(cubicCwnd)) / float64(cubicCwnd) * float64(c.maxSegmentSize))
	}

	c.cwndInc += cubicCwnd - c.cwnd
	if c.cwndInc >= c.maxSegmentSize {
		c.cwnd += c.maxSegmentSize
		c.cwndInc = 0
		c.logger.Log("congestion_window_increase", "cause", "congestion_avoidance", "window", c.cwnd)
	}
}

func (c *cubic) onCongestionEvent(_ time.Time, _ time.Time) {
	if float64(c.cwnd) < c.wMax {
		c.wMax = float64(c.cwnd) * (1.0 - cubicBeta) / 2.0
	} else {
		c.wMax = float64(c.cwnd)
	}
	c.ssthres = max(uint64(c.wMax*cubicBeta), minimumWindow(c.maxSegmentSize))
	c.cwnd = c.ssthres
	c.k = cubicK(c.wMax, c.maxSegmentSize)
	c.cwndInc = uint64(float64(c.cwndInc) * cubicBeta)
	c.logger.Log("congestion_window_decrease", "window", c.cwnd)
}

func (c *cubic) setMSS(mss uint64) {
	c.maxSegmentSize = mss
	c.cwnd = max(c.cwnd, minimumWindow(mss))
}

func (c *cubic) mss() uint64 {
	return c.maxSegmentSize
}

func (c *cubic) window() uint64 {
	return c.cwnd
}
