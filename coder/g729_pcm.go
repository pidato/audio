package coder

import (
	"errors"
	"fmt"
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/g729"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/util"
	"github.com/pidato/audio/vad"
	"time"
)

var _ Coder = &g729ulaw{}

type g729pcm struct {
	to             codec.Codec
	pcm            []int16
	next           []int16
	upsampled      []int16
	upsampledState [8]int32
	stats          Stats
	dec            *g729.Decoder
	vadState       vad.State
	onFrame        OnFrame
}

func newG729PCM(to codec.Codec, ptime codec.Ptime, onFrame OnFrame) (*g729pcm, error) {
	if to == nil {
		return nil, errors.New("to codec must be set")
	}
	if onFrame == nil {
		onFrame = func(kind Frame) error {
			return nil
		}
	}
	ptime.Validate()
	u := &g729pcm{
		pcm:     util.CastBytesToInt16(pool.U8Get(160 * (int(ptime) / 10))),
		onFrame: onFrame,
		dec:     g729.NewDecoder(),
		stats: Stats{
			ptime: ptime,
		},
	}
	if ptime > 10 {
		u.next = u.pcm[:0]
	}
	switch to.SampleRate() {
	case 8000:
	case 16000:
		u.upsampled = util.CastBytesToInt16(pool.U8Get((160 * (int(ptime) / 10)) * 2))
	default:
		_ = u.Close()
		return nil, fmt.Errorf("sample size must be 8000 or 16000 and not %d", to.SampleRate())
	}
	return u, nil
}

func (g *g729pcm) From() codec.Codec { return codec.G729 }

func (g *g729pcm) To() codec.Codec { return g.to }

func (g *g729pcm) Algo() VADAlgo {
	return G729VAD
}

func (g *g729pcm) VAD() vad.State {
	return g.vadState
}

func (g *g729pcm) Ptime() codec.Ptime {
	return g.stats.ptime
}

func (g *g729pcm) Stats() *Stats {
	return &g.stats
}

func (g *g729pcm) Duration() time.Duration {
	return g.stats.Duration()
}

func (g *g729pcm) Close() error {
	if g.dec != nil {
		_ = g.dec.Close()
		g.dec = nil
	}
	if g.pcm != nil {
		pool.I16Release(g.pcm)
		g.pcm = nil
	}
	if g.upsampled != nil {
		pool.I16Release(g.upsampled)
		g.upsampled = nil
	}
	return nil
}

func (g *g729pcm) Dropped(ptime codec.Ptime) error {
	if ptime < 10 {
		return ErrPartialFrame
	}
	if err := g.flush(); err != nil {
		return err
	}
	g.vadState = vad.Noise
	g.stats.addDropped(int64(ptime))
	count := int(ptime / 10)
	var b [10]byte
	for i := 0; i < count; i++ {
		if err := g.dec.DecodeWithOptions(b[0:10], true, false, false, g.pcm); err != nil {
			return err
		}
	}
	return g.onFrame(NewFrame(FrameDropped, ptime, nil))
}

func (g *g729pcm) flush() error {
	if len(g.next) > 0 {
		frame := g.next
		if g.upsampled != nil {
			err := vad.UpSampleBy2(g.next, g.upsampled, &g.upsampledState)
			if err != nil {
				return err
			}
			frame = g.upsampled[0 : len(g.next)*2]
		}
		err := g.onFrame(NewFrame(FrameVoice, codec.Ptime(len(g.next)/8), util.CastInt16ToBytes(frame)))
		g.next = g.next[:0]
		return err
	}
	return nil
}

func (g *g729pcm) pushVoice(b []byte) error {
	g.vadState = vad.Voice
	if g.next != nil {
		next := g.next[len(g.next) : len(g.next)+80]
		g.next = g.next[0 : len(g.next)+80]
		err := g.dec.DecodeWithOptions(b, false, false, false, next)
		if err != nil {
			return err
		}
		return g.flush()
	} else {
		err := g.dec.DecodeWithOptions(b, false, false, false, g.pcm)
		if err != nil {
			return err
		}
		if g.upsampled != nil {
			err = vad.UpSampleBy2(g.pcm, g.upsampled, &g.upsampledState)
			if err != nil {
				return err
			}
			return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(g.upsampled)))
		}

		return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(g.pcm)))
	}
}

func (g *g729pcm) Transcode(kind FrameType, b []byte) error {
	frame := g.pcm
	switch len(b) {
	case 0:
		if kind == FrameDropped {
			return g.Dropped(g.stats.ptime)
		}
		g.vadState = vad.Noise
		g.stats.addNoise(int64(g.stats.ptime))
		return g.onFrame(NewFrame(FrameNoise, g.stats.ptime, nil))
	case 2:
		if err := g.flush(); err != nil {
			return err
		}
		g.stats.addComfort(10)
		if err := g.dec.DecodeWithOptions(b, false, true, false, g.pcm); err != nil {
			return err
		}
		if g.upsampled != nil {
			if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
				return err
			}
			frame = g.upsampled
		}
		return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))

	case 10:
		return g.pushVoice(b)

	default:
		for begin := 0; begin < len(b); begin += 10 {
			end := begin + 10
			if end > len(b) {
				end = len(b)
			}
			packet := b[begin:end]
			switch len(packet) {
			case 2:
				err := g.flush()
				if err != nil {
					return err
				}
				g.stats.addComfort(10)
				if err := g.dec.DecodeWithOptions(packet, false, true, false, g.pcm); err != nil {
					return err
				}
				if g.upsampled != nil {
					if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
						return err
					}
					frame = g.upsampled
				}
				return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
			case 10:
				err := g.pushVoice(packet)
				if err != nil {
					return err
				}
			default:
				return ErrInvalidG729Frame
			}
		}
		return nil
	}
}
