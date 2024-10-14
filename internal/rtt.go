package internal

import (
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

const initialRTT = time.Millisecond * 333

type RTT struct {
	minRTT      time.Duration
	latestRTT   time.Duration
	smoothedRTT time.Duration
	rttVar      time.Duration
}

func NewRTT() *RTT {
	return &RTT{
		minRTT:      -1,
		smoothedRTT: initialRTT,
		rttVar:      initialRTT / 2,
	}
}

func (r *RTT) Add(latestRTT, ackDelay time.Duration) {
	r.latestRTT = latestRTT
	if r.minRTT < 0 {
		r.minRTT = latestRTT
		r.smoothedRTT = latestRTT
		r.rttVar = latestRTT / 2
		return
	}

	r.minRTT = min(r.minRTT, latestRTT)
	adjustedRTT := latestRTT - min(ackDelay, protocol.MaxAckDelay)
	if adjustedRTT < r.minRTT {
		adjustedRTT = latestRTT
	}
	r.smoothedRTT = ((7 * r.smoothedRTT) + adjustedRTT) / 8
	r.rttVar = (3*r.rttVar + abs(r.smoothedRTT-adjustedRTT)) / 4
}

func (r *RTT) RTT() time.Duration {
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

func abs[T ~int64](a T) T {
	if a < 0 {
		return -a
	}
	return a
}
