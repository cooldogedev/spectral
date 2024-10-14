package internal

import "errors"

type RingBuffer[T any] struct {
	data   []T
	size   int
	start  int
	length int
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, size),
		size: size,
	}
}

func (r *RingBuffer[T]) Write(p []T) (int, error) {
	if len(p) > r.size-r.length {
		return 0, errors.New("insufficient space")
	}

	for i := 0; i < len(p); i++ {
		r.data[(r.start+r.length)%r.size] = p[i]
		r.length++
	}
	return len(p), nil
}

func (r *RingBuffer[T]) Read(p []T) int {
	n := len(p)
	if n > r.length {
		n = r.length
	}

	for i := 0; i < n; i++ {
		p[i] = r.data[(r.start+i)%r.size]
	}
	r.start = (r.start + n) % r.size
	r.length -= n
	return n
}

func (r *RingBuffer[T]) Len() int {
	return r.length
}

func (r *RingBuffer[T]) Free() int {
	if len(r.data) == 0 {
		return 0
	}
	return r.size - r.length
}

func (r *RingBuffer[T]) Reset() {
	r.data = r.data[:0]
	r.data = nil
	r.length = 0
	r.size = 0
	r.start = 0
}
