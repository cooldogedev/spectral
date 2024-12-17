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
	pacer             *pacer
}

func NewSender(logger log.Logger, now time.Time, mss uint64) *Sender {
	return &Sender{
		recoveryStartTime: now,
		cc:                newReno(logger, mss),
		pacer:             newPacer(now),
	}
}

func (s *Sender) TimeUntilSend(now time.Time, rtt *RTT, bytes uint64) time.Time {
	return s.pacer.timeUntilSend(now, rtt.SRTT(), bytes, s.cc.mss(), s.cc.window())
}

func (s *Sender) OnSend(bytes uint64) {
	s.flight += bytes
	s.pacer.onSend(bytes)
	if s.recoverySend {
		s.recoverySend = false
	}
}

func (s *Sender) OnAck(now, sent time.Time, rtt *RTT, bytes uint64) {
	if s.flight > bytes {
		s.flight -= bytes
	} else {
		s.flight = 0
	}
	s.cc.onAck(now, sent, s.recoveryStartTime, rtt, bytes, s.flight)
}

func (s *Sender) OnCongestionEvent(now time.Time, sent time.Time) {
	if sent.After(s.recoveryStartTime) {
		s.recoverySend = true
		s.recoveryStartTime = now
		s.cc.onCongestionEvent(now, sent)
	}
}

func (s *Sender) SetMSS(mss uint64) {
	s.cc.setMSS(mss)
}

func (s *Sender) Available() uint64 {
	if s.recoverySend {
		return s.cc.mss()
	}

	if window := s.cc.window(); window > s.flight {
		return window - s.flight
	}
	return 0
}
