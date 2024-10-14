package spectral

import (
	"github.com/cooldogedev/spectral/internal/protocol"
	"time"
)

const (
	mtuMin  = protocol.MinPacketSize
	mtuMax  = protocol.MaxPacketSize
	mtuDiff = 20
)

const (
	probeDelay    = 5
	probeAttempts = 3
)

type mtuDiscovery struct {
	mtuIncrease func(mtu uint64)
	flight      int
	current     uint64
	discovered  bool
	prev        time.Time
}

func newMTUDiscovery(now time.Time, mtuIncrease func(mtu uint64)) *mtuDiscovery {
	m := &mtuDiscovery{
		mtuIncrease: mtuIncrease,
		current:     mtuMin,
		prev:        now,
	}
	m.discover()
	return m
}

func (m *mtuDiscovery) onAck(mtu uint64) {
	if m.current != mtu || m.mtuIncrease == nil {
		return
	}

	m.mtuIncrease(m.current)
	if !m.discovered {
		m.discover()
	}
}

func (m *mtuDiscovery) sendProbe(now time.Time, rtt time.Duration) bool {
	if now.Sub(m.prev) < rtt*probeDelay {
		return false
	}

	if m.flight >= probeAttempts {
		m.discovered = true
		return false
	}
	m.flight++
	m.prev = now
	return true
}

func (m *mtuDiscovery) discover() {
	if m.current >= mtuMax {
		m.discovered = true
		return
	}
	m.flight = 0
	m.current = min(m.current+mtuDiff, mtuMax)
}
