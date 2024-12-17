package congestion

import (
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type controller interface {
	onAck(now, sent, recoveryStartTime time.Time, rtt *RTT, bytes, flight uint64)
	onCongestionEvent(now, sent time.Time)
	setMSS(mss uint64)
	mss() uint64
	window() uint64
}

func initialWindow(mss uint64) uint64 {
	return clamp(14720, 2*mss, 10*mss)
}

func minimumWindow(mss uint64) uint64 {
	return 2 * mss
}

func shouldIncreaseWindow(flight, window, ssthres uint64) bool {
	if flight >= window {
		return true
	}
	availableBytes := window - flight
	slowStartLimited := ssthres > window && flight > window/2
	return slowStartLimited || availableBytes <= 3*protocol.MaxPacketSize
}

func clamp(value, min, max uint64) uint64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}
