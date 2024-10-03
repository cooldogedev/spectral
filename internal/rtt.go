package internal

import (
	"sync/atomic"
	"time"
)

const (
	DefaultRTT = time.Millisecond * 100

	alpha         = 0.125
	alphaMinusOne = 1.0 - alpha
)

type RTT struct {
	rtt atomic.Int64
}

func NewRTT() *RTT {
	return &RTT{}
}

func (r *RTT) Add(rtt time.Duration) {
	if rtt <= 0 {
		return
	}

	currentRTT := float64(r.rtt.Load())
	if currentRTT == 0 {
		r.rtt.Store(int64(rtt))
	} else {
		r.rtt.Store(int64(alpha*float64(rtt) + alphaMinusOne*currentRTT))
	}
}

func (r *RTT) RTT() time.Duration {
	return time.Duration(r.rtt.Load())
}
