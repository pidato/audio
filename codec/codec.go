package codec

import (
	"time"
)

var (
	ULAW  ulaw
	ALAW  alaw
	PCM8  pcm8
	PCM16 pcm16
	G729  g729codec
)

type Ptime int64

func (p Ptime) Duration() time.Duration {
	return time.Duration(p) * time.Millisecond
}

type Codec interface {
	PayloadType() uint8

	Name() string

	SampleRate() int32
}

func NewCodec(pt uint8) Codec {
	switch pt {
	case 0:
		return ULAW
	case 8:
		return ALAW
	case 18:
		return G729
	}
	return nil
}
