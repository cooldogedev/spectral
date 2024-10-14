package congestion

import (
	"math"
	"time"

	"github.com/cooldogedev/spectral/internal/log"
)

const renoReductionFactor = 0.5

type reno struct {
	mss        uint64
	window     uint64
	ssthres    uint64
	bytesAcked uint64
	logger     log.Logger
}

func newReno(logger log.Logger, mss uint64) *reno {
	return &reno{
		mss:     mss,
		window:  initialWindow(mss),
		ssthres: math.MaxUint64,
		logger:  logger,
	}
}

func (r *reno) OnAck(_, _, _ time.Time, _ time.Duration, bytes uint64) {
	if r.window < r.ssthres {
		r.window += bytes
		r.logger.Log("congestion_window_increase", "cause", "slow_start", "window", r.window)
		if r.window >= r.ssthres {
			r.bytesAcked = r.window - r.ssthres
			r.logger.Log("congestion_exit_slow_start", "window", r.window, "threshold", r.ssthres)
		}
		return
	}

	r.bytesAcked += bytes
	if r.bytesAcked >= r.window {
		r.bytesAcked -= r.window
		r.window += r.mss
		r.logger.Log("congestion_window_increase", "cause", "congestion_avoidance", "window", r.window, "acked", r.bytesAcked)
	}
}

func (r *reno) OnCongestionEvent(_ time.Time, _ time.Time) {
	r.window = uint64(float64(r.window) * renoReductionFactor)
	r.window = max(r.window, minimumWindow(r.mss))
	r.ssthres = r.window
	r.logger.Log("congestion_window_decrease", "window", r.window)
}

func (r *reno) SetMSS(mss uint64) {
	r.mss = mss
	r.window = max(r.window, minimumWindow(mss))
}

func (r *reno) MSS() uint64 {
	return r.mss
}

func (r *reno) Window() uint64 {
	return r.window
}
