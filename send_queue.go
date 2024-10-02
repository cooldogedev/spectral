package spectral

import (
	"sync"

	"github.com/cooldogedev/spectral/internal/frame"
	"github.com/cooldogedev/spectral/internal/protocol"
)

const packetSize = protocol.MaxPacketSize - protocol.PacketHeaderSize

type sendQueue struct {
	connectionID protocol.ConnectionID
	sequenceID   uint32
	pk           []byte
	list         [][]byte
	cond         *sync.Cond
	mu           sync.RWMutex
}

func newSendQueue(connectionID protocol.ConnectionID) *sendQueue {
	s := &sendQueue{
		connectionID: connectionID,
		sequenceID:   1,
		pk:           make([]byte, 0, packetSize),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *sendQueue) add(p []byte) {
	s.mu.Lock()
	s.list = append(s.list, p)
	s.mu.Unlock()
	s.cond.Signal()
}

func (s *sendQueue) shift() (uint32, []byte) {
	s.mu.Lock()
	defer func() {
		s.pk = s.pk[:0]
		s.sequenceID++
		s.mu.Unlock()
	}()

	var total uint32
	for len(s.list) > 0 {
		entry := s.list[0]
		if len(s.pk)+len(entry) > packetSize {
			break
		}
		s.list[0] = nil
		s.list = s.list[1:]
		s.pk = append(s.pk, entry...)
		total++
	}
	return s.sequenceID, frame.Pack(s.connectionID, s.sequenceID, total, s.pk)
}
