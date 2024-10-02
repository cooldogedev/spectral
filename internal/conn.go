package internal

import "net"

type Conn struct {
	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
	closable   bool
}

func NewConn(conn *net.UDPConn, remoteAddr *net.UDPAddr, closable bool) *Conn {
	return &Conn{
		conn:       conn,
		remoteAddr: remoteAddr,
		closable:   closable,
	}
}

func (c *Conn) Write(p []byte) (int, error) {
	return c.conn.WriteToUDP(p, c.remoteAddr)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *Conn) Close() error {
	if c.closable {
		return c.conn.Close()
	}
	return nil
}
