package frame

import (
	"fmt"
	"sync"
)

var (
	acknowledgementPool = sync.Pool{
		New: func() any {
			return &Acknowledgement{Ranges: make([]AcknowledgementRange, 0, 16)}
		},
	}
	streamDataPool = sync.Pool{
		New: func() any {
			return &StreamData{Payload: make([]byte, 0, 128)}
		},
	}
	Pool = sync.Pool{
		New: func() any {
			return make([]Frame, 0, 64)
		},
	}
)

func GetFrame(id uint32) (Frame, error) {
	switch id {
	case IDAcknowledgement:
		return acknowledgementPool.Get().(*Acknowledgement), nil
	case IDConnectionRequest:
		return &ConnectionRequest{}, nil
	case IDConnectionResponse:
		return &ConnectionResponse{}, nil
	case IDConnectionClose:
		return &ConnectionClose{}, nil
	case IDStreamClose:
		return &StreamClose{}, nil
	case IDStreamData:
		return streamDataPool.Get().(*StreamData), nil
	case IDStreamRequest:
		return &StreamRequest{}, nil
	case IDStreamResponse:
		return &StreamResponse{}, nil
	default:
		return nil, fmt.Errorf("unknown frame: %v", id)
	}
}

func PutFrame(fr Frame) {
	fr.Reset()
	switch fr.ID() {
	case IDAcknowledgement:
		acknowledgementPool.Put(fr)
	case IDStreamData:
		streamDataPool.Put(fr)
	default:
	}
}
