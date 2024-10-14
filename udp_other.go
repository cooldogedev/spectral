//go:build !linux && !windows

package spectral

import "syscall"

func setOpts(conn syscall.RawConn) (mtud, ecn bool) {
	return
}

func isSendMsgSizeErr(err error) bool {
	return false
}

func isRecvMsgSizeErr(err error) bool {
	return false
}
