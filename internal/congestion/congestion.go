package congestion

import "time"

type controller interface {
	OnAck(now, sent, recoveryTime time.Time, rtt time.Duration, bytes uint64)
	OnCongestionEvent(now time.Time, sent time.Time)
	SetMSS(mss uint64)
	MSS() uint64
	Window() uint64
}

func initialWindow(mss uint64) uint64 {
	return clamp(14720, 2*mss, 10*mss)
}

func minimumWindow(mss uint64) uint64 {
	return 2 * mss
}

func clamp(value, min, max uint64) uint64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}
