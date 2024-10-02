package internal

import (
	"sync/atomic"
	"time"
)

const (
	alpha         = 0.1
	alphaMinusOne = 1.0 - alpha
	defaultRTT    = time.Millisecond * 100
)

type RTT struct {
	rtt atomic.Int64
}

func NewRTT() *RTT {
	r := &RTT{}
	r.rtt.Store(int64(defaultRTT))
	return r
}

func (c *RTT) Add(rtt time.Duration) {
	currentRTT := float64(c.rtt.Load())
	c.rtt.Store(int64(alpha*float64(rtt) + alphaMinusOne*currentRTT))
}

func (c *RTT) RTT() time.Duration {
	return time.Duration(c.rtt.Load())
}
