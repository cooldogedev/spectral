//go:build windows

package spectral

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

const IP_DONTFRAGMENT = 14

func setOpts(conn syscall.RawConn) (mtud, ecn bool) {
	_ = conn.Control(func(fd uintptr) {
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, IP_DONTFRAGMENT, 1); err == nil {
			mtud = true
		}
	})
	return
}

func isSendMsgSizeErr(err error) bool {
	return errors.Is(err, windows.WSAEMSGSIZE)
}

func isRecvMsgSizeErr(err error) bool {
	return errors.Is(err, windows.WSAEMSGSIZE)
}
