package coder

import (
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/vad"
	"time"
)

var _ Coder = &g729g729{}

type g729g729 struct {
	next     []byte
	stats    Stats
	vadState vad.State
	onFrame  OnFrame
}

func newG729G729(ptime codec.Ptime, onFrame OnFrame) (*g729g729, error) {
	if onFrame == nil {
		onFrame = func(kind Frame) error {
			return nil
		}
	}
	ptime.Validate()
	g := &g729g729{
		stats: Stats{
			ptime: ptime,
		},
		onFrame: onFrame,
	}
	if ptime > 10 {
		g.next = pool.U8Get(int(ptime))
	}
	return g, nil
}

func (u *g729g729) From() codec.Codec { return codec.G729 }

func (u *g729g729) To() codec.Codec { return codec.G729 }

func (u *g729g729) Algo() VADAlgo {
	return G729VAD
}

func (u *g729g729) VAD() vad.State {
	return u.vadState
}

func (u *g729g729) Ptime() codec.Ptime {
	return u.stats.ptime
}

func (u *g729g729) Stats() *Stats {
	return &u.stats
}

func (u *g729g729) Duration() time.Duration {
	return u.stats.Duration()
}

func (g *g729g729) Close() error {
	if g.next != nil {
		pool.U8Release(g.next)
		g.next = nil
	}
	return nil
}

func (u *g729g729) Dropped(ptime codec.Ptime) error {
	u.vadState = vad.Noise
	u.stats.addDropped(int64(u.stats.ptime))
	return u.onFrame(NewFrame(FrameDropped, ptime, nil))
}

func (g *g729g729) Transcode(kind FrameType, b []byte) error {
	switch kind {
	case FrameDropped:
		return g.Dropped(g.stats.ptime)
	case FrameNoise:
		g.vadState = vad.Noise
		g.stats.addNoise(int64(g.stats.ptime))
		return g.onFrame(NewFrame(FrameDropped, g.stats.ptime, b))
	case FrameVoice:
		fallthrough
	default:
		if g.next != nil {
			if len(b) == 10 {
				g.vadState = vad.Voice
				g.next = append(g.next, b...)
				if len(g.next) == int(g.stats.ptime) {
					g.stats.addVoice(int64(g.stats.ptime))
					err := g.onFrame(NewFrame(FrameVoice, g.stats.ptime, g.next))
					g.next = g.next[:0]
					return err
				}
				return nil
			}

			g.vadState = vad.Voice
			for i := 0; i < len(b); i += 10 {
				g.next = append(g.next, b[i:i+10]...)
				if len(g.next) == int(g.stats.ptime) {
					g.stats.addVoice(int64(g.stats.ptime))
					if err := g.onFrame(NewFrame(FrameVoice, g.stats.ptime, g.next)); err != nil {
						return err
					}
					g.next = g.next[:0]
				}
			}

			switch len(b) % 10 {
			case 0:
			case 2:
				g.vadState = vad.Noise
				if err := g.onFrame(NewFrame(FrameNoise, 10, b[len(b)-2:])); err != nil {
					return err
				}
			default:
				return ErrInvalidG729Frame
			}

			return nil
		}
		if len(b) == 10 {
			g.vadState = vad.Voice
			g.stats.addVoice(10)
			return g.onFrame(NewFrame(FrameVoice, 10, b))
		}
		g.vadState = vad.Voice
		for i := 0; i < len(b); i += 10 {
			g.stats.addVoice(10)
			if err := g.onFrame(NewFrame(FrameVoice, 10, b[i:i+10])); err != nil {
				return err
			}
		}
		switch len(b) % 10 {
		case 0:
		case 2:
			g.vadState = vad.Noise
			if err := g.onFrame(NewFrame(FrameNoise, 10, b[len(b)-2:])); err != nil {
				return err
			}
		default:
			return ErrInvalidG729Frame
		}
	}
	return nil
}
