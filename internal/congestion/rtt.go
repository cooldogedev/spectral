package congestion

import (
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const initialRTT = time.Millisecond * 333

type RTT struct {
	measured    bool
	minRTT      time.Duration
	latestRTT   time.Duration
	smoothedRTT time.Duration
	rttVar      time.Duration
}

func NewRTT() *RTT {
	return &RTT{
		minRTT:      initialRTT,
		smoothedRTT: initialRTT,
		rttVar:      initialRTT / 2,
	}
}

func (r *RTT) Add(latestRTT, ackDelay time.Duration) {
	if latestRTT <= 0 {
		return
	}

	r.latestRTT = latestRTT
	if r.measured {
		r.measured = true
		r.minRTT = latestRTT
		r.smoothedRTT = latestRTT
		r.rttVar = latestRTT / 2
		return
	}

	r.minRTT = min(r.minRTT, latestRTT)
	ackDelay = min(ackDelay, protocol.MaxAckDelay)
	adjustedRTT := latestRTT
	if latestRTT >= r.minRTT+ackDelay {
		adjustedRTT -= ackDelay
	}

	var rttSample time.Duration
	if r.smoothedRTT > adjustedRTT {
		rttSample = r.smoothedRTT - adjustedRTT
	} else {
		rttSample = adjustedRTT - r.smoothedRTT
	}
	r.rttVar = ((3 * r.rttVar) + rttSample) / 4
	r.smoothedRTT = ((7 * r.smoothedRTT) + adjustedRTT) / 8
}

func (r *RTT) LatestRTT() time.Duration {
	return r.latestRTT
}

func (r *RTT) SRTT() time.Duration {
	return r.smoothedRTT
}

func (r *RTT) RTTVAR() time.Duration {
	return r.rttVar
}

func (r *RTT) RTO() time.Duration {
	return r.smoothedRTT + max(4*r.rttVar, protocol.TimerGranularity) + protocol.MaxAckDelay
}
