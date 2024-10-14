package spectral

import (
	"net"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type udpConn struct {
	conn     *net.UDPConn
	ecn      bool
	mtud     bool
	listener bool
}

func newUDPConn(conn *net.UDPConn, listener bool) (*udpConn, error) {
	if err := conn.SetReadBuffer(protocol.ReceiveBufferSize); err != nil {
		return nil, err
	}

	if err := conn.SetWriteBuffer(protocol.SendBufferSize); err != nil {
		return nil, err
	}

	sc, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}
	c := &udpConn{conn: conn, listener: listener}
	c.mtud, c.ecn = setOpts(sc)
	return c, nil
}

func (c *udpConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *udpConn) Read(f func(d *datagram) error) {
	for {
		dgram := newDatagram()
		n, addr, err := c.conn.ReadFromUDP(dgram.b)
		if err != nil && !isRecvMsgSizeErr(err) {
			return
		}

		if n == 0 {
			continue
		}

		dgram.peerAddr = addr
		if err := f(dgram); err != nil {
			return
		}
	}
}

func (c *udpConn) Write(p []byte, addr *net.UDPAddr) (int, error) {
	n, err := c.conn.WriteToUDP(p, addr)
	if err != nil && !isSendMsgSizeErr(err) {
		return 0, err
	}
	return n, nil
}

func (c *udpConn) Close() error {
	if c.listener {
		return nil
	}
	return c.conn.Close()
}
