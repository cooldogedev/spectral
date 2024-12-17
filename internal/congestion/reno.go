package congestion

import (
	"math"
	"time"

	"github.com/cooldogedev/spectral/internal/log"
)

const renoReductionFactor = 0.5

type reno struct {
	maxSegmentSize uint64
	cwnd           uint64
	ssthres        uint64
	bytesAcked     uint64
	logger         log.Logger
}

func newReno(logger log.Logger, mss uint64) *reno {
	return &reno{
		maxSegmentSize: mss,
		cwnd:           initialWindow(mss),
		ssthres:        math.MaxUint64,
		logger:         logger,
	}
}

func (r *reno) onAck(_, _, _ time.Time, _ *RTT, bytes, flight uint64) {
	if !shouldIncreaseWindow(flight, r.cwnd, r.ssthres) {
		return
	}

	if r.cwnd < r.ssthres {
		r.cwnd += r.maxSegmentSize
		r.logger.Log("congestion_window_increase", "cause", "slow_start", "window", r.cwnd, "threshold", r.ssthres)
		if r.cwnd >= r.ssthres {
			r.ssthres = r.cwnd
			r.logger.Log("congestion_exit_slow_start", "window", r.cwnd, "threshold", r.ssthres)
		}
	} else {
		r.bytesAcked += bytes
		if r.bytesAcked >= r.cwnd {
			r.bytesAcked -= r.cwnd
			r.cwnd += r.maxSegmentSize
			r.logger.Log("congestion_window_increase", "cause", "congestion_avoidance", "window", r.cwnd, "threshold", r.ssthres, "acked", r.bytesAcked)
		}
	}
}

func (r *reno) onCongestionEvent(_ time.Time, _ time.Time) {
	r.cwnd = uint64(float64(r.cwnd) * renoReductionFactor)
	r.cwnd = max(r.cwnd, minimumWindow(r.maxSegmentSize))
	r.bytesAcked = uint64(float64(r.cwnd) * renoReductionFactor)
	r.ssthres = r.cwnd
	r.logger.Log("congestion_window_decrease", "window", r.cwnd, "threshold", r.ssthres)
}

func (r *reno) setMSS(mss uint64) {
	r.maxSegmentSize = mss
	r.cwnd = max(r.cwnd, minimumWindow(mss))
}

func (r *reno) mss() uint64 {
	return r.maxSegmentSize
}

func (r *reno) window() uint64 {
	return r.cwnd
}
