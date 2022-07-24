package rtp

import (
	"errors"
	"github.com/pidato/audio/pool"
	"github.com/pidato/disruptor/ring"
)

var (
	ErrFull = errors.New("full")
)

type BufferRing struct {
	ring *ring.Ring[BufferSlot]
	seq  int64
}

func (br *BufferRing) Close() error {
	c := br.ring.Cap()
	for i := int64(0); i < c; i++ {
		slot := br.ring.Get(i)
		if slot.B != nil {
			pool.U8Release(slot.B)
			slot.B = nil
		}
	}
	return nil
}

func (br *BufferRing) Next() (*BufferSlot, error) {
	slot := br.ring.Get(br.seq)
	if slot.B != nil {
		return nil, ErrFull
	}
	br.seq++
	return slot, nil
}

type BufferSlot struct {
	B []byte
}

func (b *BufferSlot) Alloc(size int) []byte {
	return pool.U8Get(size)
}

func (b *BufferSlot) Take() []byte {
	r := b.B
	b.B = nil
	return r
}
