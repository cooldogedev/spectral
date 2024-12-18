package frame

import (
	"fmt"
	"sync"
)

var (
	acknowledgementPool = sync.Pool{
		New: func() any {
			return &Acknowledgement{Ranges: make([]AcknowledgementRange, 0, 128)}
		},
	}
	streamDataPool = sync.Pool{
		New: func() any {
			return &StreamData{Payload: make([]byte, 0, 1452)}
		},
	}
)

func getFrame(id uint32) (Frame, error) {
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
	case IDMTURequest:
		return &MTURequest{}, nil
	case IDMTUResponse:
		return &MTUResponse{}, nil
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
