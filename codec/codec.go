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

func (p *Ptime) Validate() bool {
	v := int64(*p)
	if v < 10 {
		*p = 10
		return false
	}
	m := v % 10
	if m == 0 {
		if v > 100 {
			*p = 100
			return false
		}
		return true
	}
	v += m
	if v > 100 {
		*p = 100
		return false
	}
	*p = Ptime(v)
	return false
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
