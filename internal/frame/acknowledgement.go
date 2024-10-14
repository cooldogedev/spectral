package frame

import (
	"encoding/binary"
	"errors"
	"slices"
)

type AcknowledgementRange [2]uint32

type Acknowledgement struct {
	Delay  int64
	Max    uint32
	Ranges []AcknowledgementRange
}

func (fr *Acknowledgement) ID() uint32 {
	return IDAcknowledgement
}

func (fr *Acknowledgement) Encode() []byte {
	rangeCount := uint32(len(fr.Ranges))
	p := make([]byte, 8+4+4+rangeCount*8)
	binary.LittleEndian.PutUint64(p[0:8], uint64(fr.Delay))
	binary.LittleEndian.PutUint32(p[8:12], fr.Max)
	binary.LittleEndian.PutUint32(p[12:16], rangeCount)
	for i, r := range fr.Ranges {
		offset := 16 + i*8
		binary.LittleEndian.PutUint32(p[offset:offset+4], r[0])
		binary.LittleEndian.PutUint32(p[offset+4:offset+8], r[1])
	}
	return p
}

func (fr *Acknowledgement) Decode(p []byte) (int, error) {
	if len(p) < 16 {
		return 0, errors.New("not enough data to decode")
	}

	fr.Delay = int64(binary.LittleEndian.Uint64(p[0:8]))
	fr.Max = binary.LittleEndian.Uint32(p[8:12])
	length := binary.LittleEndian.Uint32(p[12:16])
	if len(p) < 16+int(length)*8 {
		return 0, errors.New("not enough data to decode ranges")
	}

	fr.Ranges = fr.Ranges[:length]
	for i := uint32(0); i < length; i++ {
		offset := 16 + i*8
		fr.Ranges[i][0] = binary.LittleEndian.Uint32(p[offset : offset+4])
		fr.Ranges[i][1] = binary.LittleEndian.Uint32(p[offset+4 : offset+8])
	}
	return 12 + int(length)*8, nil
}

func (fr *Acknowledgement) Reset() {
	fr.Delay = 0
	fr.Max = 0
	fr.Ranges = fr.Ranges[:0]
}

func GenerateAcknowledgementRanges(list []uint32) (ranges []AcknowledgementRange) {
	slices.Sort(list)
	start := list[0]
	end := list[0]
	for _, value := range list[1:] {
		if value != end+1 {
			ranges = append(ranges, AcknowledgementRange{start, end})
			start = value
		}
		end = value
	}
	ranges = append(ranges, AcknowledgementRange{start, end})
	return
}
