package spectral

import (
	"net"
	"sync"

	"github.com/cooldogedev/spectral/internal/protocol"
)

var datagramPool = sync.Pool{
	New: func() any {
		return &datagram{b: make([]byte, protocol.MaxUDPPayloadSize)}
	},
}

type datagram struct {
	b        []byte
	peerAddr *net.UDPAddr
}

func newDatagram() *datagram {
	dgram := datagramPool.Get().(*datagram)
	*dgram = datagram{b: dgram.b[:cap(dgram.b)]}
	return dgram
}

func (d *datagram) reset() {
	if cap(d.b) == protocol.MaxUDPPayloadSize {
		datagramPool.Put(d)
	}
}
