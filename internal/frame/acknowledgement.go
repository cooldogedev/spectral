package frame

import (
	"encoding/binary"
	"errors"
)

const (
	AcknowledgementWithGaps byte = iota
	AcknowledgementWithoutGaps
)

type AcknowledgementRange [2]uint32

type Acknowledgement struct {
	Type   byte
	Delay  int64
	Ranges []AcknowledgementRange
}

func (fr *Acknowledgement) ID() uint32 {
	return IDAcknowledgement
}

func (fr *Acknowledgement) Encode() ([]byte, error) {
	rangeCount := uint32(len(fr.Ranges))
	p := make([]byte, 1+8+4+rangeCount*8)
	p[0] = fr.Type
	binary.LittleEndian.PutUint64(p[1:9], uint64(fr.Delay))
	binary.LittleEndian.PutUint32(p[9:13], rangeCount)
	for i, r := range fr.Ranges {
		offset := 13 + i*8
		binary.LittleEndian.PutUint32(p[offset:offset+4], r[0])
		binary.LittleEndian.PutUint32(p[offset+4:offset+8], r[1])
	}
	return p, nil
}

func (fr *Acknowledgement) Decode(p []byte) (int, error) {
	if len(p) < 12 {
		return 0, errors.New("not enough data to decode")
	}

	fr.Type = p[0]
	fr.Delay = int64(binary.LittleEndian.Uint64(p[1:9]))
	length := binary.LittleEndian.Uint32(p[9:13])
	if len(p) < 12+int(length)*8 {
		return 0, errors.New("not enough data to decode ranges")
	}

	fr.Ranges = fr.Ranges[:length]
	for i := uint32(0); i < length; i++ {
		offset := 13 + i*8
		fr.Ranges[i][0] = binary.LittleEndian.Uint32(p[offset : offset+4])
		fr.Ranges[i][1] = binary.LittleEndian.Uint32(p[offset+4 : offset+8])
	}
	return 13 + int(length)*8, nil
}

func (fr *Acknowledgement) Reset() {
	fr.Type = 0
	fr.Delay = 0
	fr.Ranges = fr.Ranges[:0]
}

func GenerateAcknowledgementRange(list []uint32) (byte, []AcknowledgementRange) {
	var ranges []AcknowledgementRange
	ackType := AcknowledgementWithoutGaps
	start := list[0]
	end := list[0]
	for _, value := range list[1:] {
		if value != end+1 {
			ackType = AcknowledgementWithGaps
			ranges = append(ranges, AcknowledgementRange{start, end})
			start = value
		}
		end = value
	}
	return ackType, append(ranges, AcknowledgementRange{start, end})
}

func GenerateAcknowledgementGaps(ranges []AcknowledgementRange) []uint32 {
	var gaps []uint32
	for i := 0; i < len(ranges)-1; i++ {
		currentEnd := ranges[i][1]
		nextStart := ranges[i+1][0]
		if nextStart > currentEnd+1 {
			for j := currentEnd + 1; j < nextStart; j++ {
				gaps = append(gaps, j)
			}
		}
	}
	return gaps
}
