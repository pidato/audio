package transcode

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

var _ Transcoder = &g729ulaw{}

type g729pcm struct {
	to             codec.Codec
	pcm            []int16
	upsampled      []int16
	upsampledState [8]int32
	stats          Stats
	dec            *g729.Decoder
	vad            *vad.VAD
	vadState       vad.State
	vadAlgo        VADAlgo
	onFrame        OnFrame
}

func newG729PCM(to codec.Codec, algo VADAlgo, onFrame OnFrame) (*g729pcm, error) {
	if to == nil {
		return nil, errors.New("to codec must be set")
	}
	if onFrame == nil {
		onFrame = func(kind Frame) error {
			return nil
		}
	}
	u := &g729pcm{
		pcm:     pool.I16Get(80),
		onFrame: onFrame,
		vadAlgo: algo,
		dec:     g729.NewDecoder(),
		stats: Stats{
			ptime: 10,
		},
	}
	switch to.SampleRate() {
	case 8000:
	case 16000:
		u.upsampled = pool.I16Get(160)
	default:
		_ = u.Close()
		return nil, fmt.Errorf("sample size must be 8000 or 16000 and not %d", to.SampleRate())
	}
	switch algo {
	case NoVAD:
	case WebRTCQuality:
		u.vad, _ = vad.New(8000, vad.Quality)
	case WebRTCLowBitrate:
		u.vad, _ = vad.New(8000, vad.LowBitrate)
	case WebRTCAggressive:
		u.vad, _ = vad.New(8000, vad.Aggressive)
	case WebRTCVeryAggressive:
		u.vad, _ = vad.New(8000, vad.VeryAggressive)
	default:
		u.vadAlgo = WebRTCLowBitrate
		u.vad, _ = vad.New(8000, vad.LowBitrate)
	}
	return u, nil
}

func (u *g729pcm) From() codec.Codec { return codec.G729 }

func (u *g729pcm) To() codec.Codec { return u.to }

func (u *g729pcm) Algo() VADAlgo {
	return u.vadAlgo
}

func (u *g729pcm) VAD() vad.State {
	return u.vadState
}

func (u *g729pcm) Ptime() codec.Ptime {
	return u.stats.ptime
}

func (u *g729pcm) Stats() *Stats {
	return &u.stats
}

func (u *g729pcm) Duration() time.Duration {
	return u.stats.Duration()
}

func (g *g729pcm) Close() error {
	if g.dec != nil {
		_ = g.dec.Close()
		g.dec = nil
	}
	if g.vad != nil {
		_ = g.vad.Close()
		g.vad = nil
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

func (u *g729pcm) Dropped(ptime codec.Ptime) error {
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

func (g *g729pcm) Transcode(kind FrameType, b []byte) error {
	frame := g.pcm
	switch len(b) {
	case 0:
		g.stats.addFrames(1)
		g.stats.addNoise(1)
		if kind != FrameDropped {
			if err := g.dec.DecodeWithOptions(b, true, false, false, g.pcm); err != nil {
				return err
			}

			if g.upsampled != nil {
				if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
					return err
				}
				frame = g.upsampled
			}

			if g.vad != nil {
				g.vadState = g.vad.Process(g.pcm)
				if g.vadState == vad.Voice {
					return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame)))
				} else {
					return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
				}
			} else {
				return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
			}
		} else {
			return g.onFrame(NewFrame(FrameDropped, 10, b))
		}
	case 2:
		g.stats.addFrames(1)
		g.stats.addComfort(1)
		if err := g.dec.DecodeWithOptions(b, false, true, false, g.pcm); err != nil {
			return err
		}

		if g.upsampled != nil {
			if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
				return err
			}
			frame = g.upsampled
		}

		if g.vad != nil {
			g.vadState = g.vad.Process(g.pcm)
			if g.vadState == vad.Voice {
				return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame)))
			} else {
				return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
			}
		} else {
			return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
		}
	case 10:
		g.stats.addFrames(1)
		g.stats.addVoice(1)
		if err := g.dec.DecodeWithOptions(b, false, false, false, g.pcm); err != nil {
			return err
		}

		if g.upsampled != nil {
			if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
				return err
			}
			frame = g.upsampled
		}

		if g.vad != nil {
			g.vadState = g.vad.Process(g.pcm)
			if g.vadState == vad.Voice {
				return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame)))
			} else {
				return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
			}
		} else {
			return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame)))
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
				g.stats.addComfort(1)
				if err := g.dec.DecodeWithOptions(packet, false, true, false, g.pcm); err != nil {
					return err
				}

				if g.upsampled != nil {
					if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
						return err
					}
					frame = g.upsampled
				}

				if g.vad != nil {
					g.vadState = g.vad.Process(g.pcm)
					if g.vadState == vad.Voice {
						return g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame)))
					} else {
						return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
					}
				} else {
					return g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame)))
				}
			case 10:
				if err := g.dec.DecodeWithOptions(packet, false, false, false, g.pcm); err != nil {
					return err
				}

				if g.upsampled != nil {
					if err := vad.UpSampleBy2(frame, g.upsampled, &g.upsampledState); err != nil {
						return err
					}
					frame = g.upsampled
				}

				if g.vad != nil {
					g.vadState = g.vad.Process(g.pcm)
					if g.vadState == vad.Voice {
						g.stats.addVoice(1)
						if err := g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame))); err != nil {
							return err
						}
					} else {
						g.stats.addNoise(1)
						if err := g.onFrame(NewFrame(FrameNoise, 10, util.CastInt16ToBytes(frame))); err != nil {
							return err
						}
					}
				} else {
					g.stats.addVoice(1)
					if err := g.onFrame(NewFrame(FrameVoice, 10, util.CastInt16ToBytes(frame))); err != nil {
						return err
					}
				}
			default:
				return ErrInvalidG729Frame
			}
		}
		return nil
	}
}
