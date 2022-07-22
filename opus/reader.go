package opus

import "io"

type PacketReader interface {
	io.Closer

	ReadPacket() ([]byte, error)
}

type DecodingReader interface {
	io.Closer
}
