package congestion

import (
	"time"

	"github.com/cooldogedev/spectral/internal/log"
)

type Sender struct {
	flight            uint64
	recoverySend      bool
	recoveryStartTime time.Time
	cc                controller
}

func NewRenoSender(logger log.Logger, now time.Time, mss uint64) *Sender {
	return &Sender{
		cc:                newReno(logger, mss),
		recoveryStartTime: now,
	}
}

func (r *Sender) OnSend(bytes uint64) {
	r.flight += bytes
	if r.recoverySend {
		r.recoverySend = false
	}
}

func (r *Sender) OnAck(now, sent time.Time, rtt time.Duration, bytes uint64) {
	r.flight = max(r.flight-bytes, 0)
	r.cc.OnAck(now, sent, r.recoveryStartTime, rtt, bytes)
}

func (r *Sender) OnCongestionEvent(now time.Time, sent time.Time) {
	if sent.After(r.recoveryStartTime) {
		r.recoverySend = true
		r.recoveryStartTime = now
		r.cc.OnCongestionEvent(now, sent)
	}
}

func (r *Sender) SetMSS(mss uint64) {
	r.cc.SetMSS(mss)
}

func (r *Sender) Window() uint64 {
	return r.cc.Window()
}

func (r *Sender) Available() uint64 {
	if r.recoverySend {
		return r.cc.MSS()
	}
	return r.cc.Window() - r.flight
}
