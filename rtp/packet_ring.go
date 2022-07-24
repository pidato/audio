package rtp

import (
	"github.com/pidato/audio/pool"
	"github.com/pidato/disruptor/ring"
)

type PacketRing struct {
	ring *ring.Ring[PacketSlot]
	seq  int64
}

func NewPacketRing(size int64) (*PacketRing, error) {
	r, err := ring.NewRing[PacketSlot](size)
	if err != nil {
		return nil, err
	}
	return &PacketRing{ring: r}, nil
}

func (pr *PacketRing) Close() error {
	c := pr.ring.Cap()
	for i := int64(0); i < c; i++ {
		slot := pr.ring.Get(i)
		if slot.Data != nil {
			pool.U8Release(slot.Data)
			slot.Data = nil
			slot.Payload = nil
		}
	}
	return nil
}

func (pr *PacketRing) Next() (*PacketSlot, error) {
	slot := pr.ring.Get(pr.seq)
	if slot.Data != nil {
		return nil, ErrFull
	}
	pr.seq++
	return slot, nil
}

type PacketSlot struct {
	Packet
}

func (ps *PacketSlot) Alloc(size int) *Packet {
	ps.Packet = Packet{
		Data: pool.U8Get(size),
	}
	return &ps.Packet
}

func (ps *PacketSlot) AllocCopy(b []byte) (*Packet, error) {
	ps.Packet = Packet{
		Data: pool.U8Get(len(b)),
	}
	copy(ps.Packet.Data, b)
	err := ps.Packet.Packet.Unmarshal(ps.Packet.Data)
	if err != nil {
		return &ps.Packet, err
	}
	return &ps.Packet, nil
}

func (b *PacketSlot) Take() Packet {
	r := b.Packet
	b.Packet = Packet{}
	return r
}
