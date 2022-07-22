package transcode

import (
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/g729"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/vad"
	"time"
)

var _ Transcoder = &g729g729{}

type g729g729 struct {
	frame    []int16
	stats    Stats
	dec      *g729.Decoder
	vad      *vad.VAD
	vadState vad.State
	vadAlgo  VADAlgo
	onFrame  OnFrame
}

func newG729G729(algo VADAlgo, onFrame OnFrame) (*g729g729, error) {
	if onFrame == nil {
		onFrame = func(kind Frame) error {
			return nil
		}
	}
	g := &g729g729{
		stats: Stats{
			ptime: 10,
		},
		vadAlgo: algo,
		onFrame: onFrame,
	}
	if algo != G729VAD {
		switch algo {
		case NoVAD:
		case WebRTCQuality:
			g.vad, _ = vad.New(8000, vad.Quality)
		case WebRTCLowBitrate:
			g.vad, _ = vad.New(8000, vad.LowBitrate)
		case WebRTCAggressive:
			g.vad, _ = vad.New(8000, vad.Aggressive)
		case WebRTCVeryAggressive:
			g.vad, _ = vad.New(8000, vad.VeryAggressive)
		default:
			g.vadAlgo = WebRTCLowBitrate
			g.vad, _ = vad.New(8000, vad.LowBitrate)
		}

		if g.vad != nil {
			g.frame = pool.I16Get(80)
			g.dec = g729.NewDecoder()
		}
	}
	return g, nil
}

func (u *g729g729) From() codec.Codec { return codec.G729 }

func (u *g729g729) To() codec.Codec { return codec.G729 }

func (u *g729g729) Algo() VADAlgo {
	return u.vadAlgo
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
	if g.dec != nil {
		_ = g.dec.Close()
		g.dec = nil
	}
	if g.vad != nil {
		_ = g.vad.Close()
		g.vad = nil
	}
	if g.frame != nil {
		pool.I16Release(g.frame)
		g.frame = nil
	}
	return nil
}

func (u *g729g729) Dropped(ptime codec.Ptime) error {
	if ptime == u.stats.ptime {
		u.stats.addFrames(1)
		u.stats.addDropped(1)
		return u.onFrame(NewFrame(FrameDropped, ptime, nil))
	}
	if ptime == 0 || ptime%u.stats.ptime != 0 {
		return ErrPartialFrame
	}
	count := int(ptime / u.stats.ptime)
	for i := 0; i < count; i++ {
		u.stats.addFrames(1)
		u.stats.addDropped(1)
		if err := u.onFrame(NewFrame(FrameDropped, u.stats.ptime, nil)); err != nil {
			return err
		}
	}
	return nil
}

func (g *g729g729) Transcode(kind FrameType, b []byte) error {
	if g.vad == nil {
		switch kind {
		case FrameDropped:
			g.stats.addFrames(1)
			g.stats.addDropped(1)
			return g.onFrame(NewFrame(FrameDropped, 10, b))
		case FrameNoise:
			g.vadState = vad.Noise
			g.stats.addFrames(1)
			g.stats.addNoise(1)
			return g.onFrame(NewFrame(FrameDropped, 10, b))
		case FrameVoice:
			fallthrough
		default:
			if len(b) == 10 {
				g.vadState = vad.Voice
				g.stats.addFrames(1)
				g.stats.addVoice(1)
				return g.onFrame(NewFrame(FrameVoice, 10, b))
			}
			if len(b)%10 != 0 || len(b) < 10 {
				return ErrInvalidG729Frame
			}
			g.vadState = vad.Voice
			for i := 0; i < len(b); i += 10 {
				g.stats.addFrames(1)
				g.stats.addVoice(1)
				if err := g.onFrame(NewFrame(FrameVoice, 10, b[i:i+10])); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if kind == FrameDropped {
		return g.onFrame(NewFrame(FrameDropped, 10, b))
	}

	switch len(b) {
	case 0:
		g.vadState = vad.Noise
		g.stats.addFrames(1)
		g.stats.addDropped(1)
		g.vadAlgo = G729VAD
		return g.onFrame(NewFrame(kind, 10, b))
	case 2:
		g.vadState = vad.Noise
		g.stats.addFrames(1)
		g.stats.addComfort(1)
		g.vadAlgo = G729VAD
		return g.onFrame(NewFrame(FrameNoise, 10, b))
	case 10:
		g.stats.addFrames(1)
		g.stats.addVoice(1)
		if g.vadAlgo == G729VAD {
			return g.onFrame(NewFrame(FrameVoice, 10, b))
		}
		if err := g.dec.DecodeWithOptions(b, false, false, false, g.frame); err != nil {
			return err
		}
		g.vadState = g.vad.Process(g.frame)
		if g.vadState == vad.Voice {
			return g.onFrame(NewFrame(FrameVoice, 10, b))
		} else {
			return g.onFrame(NewFrame(FrameNoise, 10, b))
		}
	default:
		for begin := 0; begin < len(b); begin += 10 {
			g.stats.addFrames(1)
			end := begin + 10
			if end > len(b) {
				end = len(b)
			}
			packet := b[begin:end]
			switch len(packet) {
			case 2:
				g.vadState = vad.Noise
				g.stats.addFrames(1)
				g.stats.addComfort(1)
				g.vadAlgo = G729VAD
				return g.onFrame(NewFrame(FrameNoise, 10, packet))
			case 10:
				if g.vadAlgo == G729VAD {
					g.stats.addVoice(1)
					g.vadState = vad.Voice
					if err := g.onFrame(NewFrame(FrameVoice, 10, packet)); err != nil {
						return err
					}
				} else {
					if err := g.dec.DecodeWithOptions(packet, false, false, false, g.frame); err != nil {
						return err
					}
					g.vadState = g.vad.Process(g.frame)
					if g.vadState == vad.Voice {
						g.stats.addVoice(1)
						if err := g.onFrame(NewFrame(FrameVoice, 10, packet)); err != nil {
							return err
						}
					} else {
						g.stats.addNoise(1)
						if err := g.onFrame(NewFrame(FrameNoise, 10, packet)); err != nil {
							return err
						}
					}
				}
			default:
				return ErrInvalidG729Frame
			}
		}
		return nil
	}
}
