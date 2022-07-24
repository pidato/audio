package rtp

import (
	"github.com/pidato/audio/coder"
	"github.com/pion/rtp"
)

type Receiver struct {
	out        coder.Coder
	rollover   int
	seq        uint16
	ptime      int
	count      int64
	head       uint16
	tail       uint16
	size       uint16
	outOfOrder int64
	in         *PacketRing
	output     *PacketRing
}

func NewReceiver() (*Receiver, error) {
	input, err := NewPacketRing(32)
	if err != nil {
		return nil, err
	}
	return &Receiver{
		in: input,
	}, nil
}

func (r *Receiver) Close() error {
	if r.out != nil {
		_ = r.out.Close()
		r.out = nil
	}
	return nil
}

func (r *Receiver) Write(b []byte) (int, error) {
	p := Packet{Data: b}
	err := p.Unmarshal(b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (r *Receiver) WritePacket(b []byte, packet rtp.Packet) error {
	//p := Packet{Packet: packet, Data: b}
	switch r.count {
	case 0:

	}
	return nil
}

func (r *Receiver) run() {

}

type Packet struct {
	rtp.Packet
	Data []byte
}
