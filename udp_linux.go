//go:build linux

package spectral

import (
	"errors"
	"syscall"

	"golang.org/x/sys/unix"
)

func setOpts(conn syscall.RawConn) (mtud, ecn bool) {
	_ = conn.Control(func(fd uintptr) {
		if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, unix.IP_PMTUDISC_DO); err == nil {
			mtud = true
		}
	})
	return
}

func isSendMsgSizeErr(err error) bool {
	return errors.Is(err, unix.EMSGSIZE)
}

func isRecvMsgSizeErr(_ error) bool {
	return false
}
