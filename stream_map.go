package spectral

import (
	"sync"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type streamMap struct {
	m  map[protocol.StreamID]*Stream
	mu sync.RWMutex
}

func newStreamMap() *streamMap {
	return &streamMap{m: make(map[protocol.StreamID]*Stream)}
}

func (s *streamMap) add(stream *Stream) {
	s.mu.Lock()
	s.m[stream.streamID] = stream
	s.mu.Unlock()
}

func (s *streamMap) get(streamID protocol.StreamID) *Stream {
	s.mu.RLock()
	stream, ok := s.m[streamID]
	s.mu.RUnlock()
	if ok {
		return stream
	}
	return nil
}

func (s *streamMap) remove(streamID protocol.StreamID) {
	s.mu.Lock()
	delete(s.m, streamID)
	s.mu.Unlock()
}

func (s *streamMap) all() []*Stream {
	s.mu.RLock()
	list := make([]*Stream, 0, len(s.m))
	for _, stream := range s.m {
		list = append(list, stream)
	}
	s.mu.RUnlock()
	return list
}
