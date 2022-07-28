package coder

import (
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/g711"
	"github.com/pidato/audio/g729"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/vad"
	"io"
	"time"
)

var _ Coder = &ulawG729{}

type ulawG729 struct {
	pcm      []int16
	in       []byte
	frame    []byte
	next     []byte
	stats    Stats
	enc      *g729.Encoder
	vadState vad.State
	onFrame  OnFrame
}

func newULAWG729(ptime codec.Ptime, onFrame OnFrame) (*ulawG729, error) {
	if onFrame == nil {
		onFrame = func(frame Frame) error {
			return nil
		}
	}
	ptime.Validate()
	u := &ulawG729{
		stats: Stats{
			ptime: ptime,
		},
		pcm:     pool.I16Get(80),
		frame:   pool.U8Get(int(ptime)),
		onFrame: onFrame,
	}
	if ptime > 10 {
		u.next = u.frame[:0]
	}
	//switch algo {
	//case G729VAD:
	//	// ignore
	//case WebRTCQuality:
	//	u.vad, _ = vad.New(8000, vad.Quality)
	//case WebRTCLowBitrate:
	//	u.vad, _ = vad.New(8000, vad.LowBitrate)
	//case WebRTCAggressive:
	//	u.vad, _ = vad.New(8000, vad.Aggressive)
	//case WebRTCVeryAggressive:
	//	u.vad, _ = vad.New(8000, vad.VeryAggressive)
	//default:
	//	u.vad, _ = vad.New(8000, vad.LowBitrate)
	//}
	u.enc = g729.NewEncoder(true)
	return u, nil
}

func (u *ulawG729) From() codec.Codec { return codec.ULAW }

func (u *ulawG729) To() codec.Codec { return codec.G729 }

func (u *ulawG729) Algo() VADAlgo {
	return G729VAD
}

func (u *ulawG729) VAD() vad.State {
	return u.vadState
}

func (u *ulawG729) Ptime() codec.Ptime {
	return u.stats.ptime
}

func (u *ulawG729) Stats() *Stats {
	return &u.stats
}

func (u *ulawG729) Duration() time.Duration {
	return u.stats.Duration()
}

func (u *ulawG729) Close() error {
	if u.enc != nil {
		_ = u.enc.Close()
		u.enc = nil
	}
	if u.pcm != nil {
		pool.I16Release(u.pcm)
		u.pcm = nil
	}
	if u.in != nil {
		pool.U8Release(u.in)
		u.in = nil
	}
	if u.frame != nil {
		pool.U8Release(u.frame)
		u.frame = nil
	}
	return nil
}

func (u *ulawG729) Dropped(ptime codec.Ptime) error {
	u.stats.addDropped(int64(ptime))
	return u.onFrame(NewFrame(FrameDropped, ptime, nil))
}

func (u *ulawG729) Transcode(kind FrameType, b []byte) error {
	if kind == FrameDropped {
		return u.Dropped(u.stats.ptime)
	}
	if len(b) == 0 {
		return io.ErrShortBuffer
	}
	// Handle bad framing
	if len(u.in) > 0 {
		if len(u.in)+len(b) > 80 {
			cutoff := 80 - len(u.in)
			u.in = append(u.in, b[0:cutoff]...)
			err := u.push(u.in)
			u.in = u.in[:0]
			b = b[cutoff:]
			if err != nil {
				return err
			}
		} else {
			u.in = append(u.in, b...)
			if len(u.in) < 80 {
				return nil
			}
			err := u.push(u.in)
			u.in = u.in[:0]
			return err
		}
	}

	for {
		if len(b) < 80 {
			if len(b) > 0 {
				if u.in == nil {
					u.in = pool.U8Get(80)
				}
				u.in = append(u.in, b...)
			}
			return nil
		}
		err := u.push(b[0:80])
		if err != nil {
			return err
		}
		b = b[80:]
	}
}

func (u *ulawG729) push(b []byte) error {
	pcm := g711.DecodeULAW(u.pcm, b)

	if u.next != nil {
		next := u.next[len(u.next) : len(u.next)+10]
		length, err := u.enc.Encode(pcm, next)
		if err != nil {
			return err
		}
		next = next[0:length]

		switch length {
		case 0:
			// Flush voice if available
			if len(u.next) > 0 {
				u.stats.addVoice(int64(u.stats.ptime))
				err := u.onFrame(NewFrame(FrameVoice, codec.Ptime(len(u.next)), u.next))
				u.next = u.next[:0]
				if err != nil {
					return err
				}
			}
			u.vadState = vad.Noise
			return u.onFrame(NewFrame(FrameNoise, 10, nil))
		case 2:
			// Flush voice if available
			if len(u.next) > 0 {
				u.stats.addVoice(int64(u.stats.ptime))
				err := u.onFrame(NewFrame(FrameVoice, codec.Ptime(len(u.next)), u.next))
				u.next = u.next[:0]
				if err != nil {
					return err
				}
			}
			u.vadState = vad.Noise
			return u.onFrame(NewFrame(FrameNoise, 10, next))

		case 10:
			u.vadState = vad.Voice
			u.next = u.next[0 : len(u.next)+10]
			if len(u.next) == int(u.stats.ptime) {
				u.stats.addVoice(int64(u.stats.ptime))
				err := u.onFrame(NewFrame(FrameVoice, codec.Ptime(len(u.next)), u.next))
				u.next = u.next[:0]
				if err != nil {
					return err
				}
			}
			return nil

		default:
			return ErrInvalidG729Frame
		}
	}

	length, err := u.enc.Encode(pcm, u.frame)
	if err != nil {
		return err
	}
	switch length {
	case 0:
		u.vadState = vad.Noise
		u.stats.addNoise(10)
		return u.onFrame(NewFrame(FrameNoise, 10, u.frame[0:0]))
	case 2:
		u.vadState = vad.Noise
		u.stats.addComfort(10)
		return u.onFrame(NewFrame(FrameNoise, 10, u.frame[0:2]))
	case 10:
		u.vadState = vad.Voice
		u.stats.addVoice(10)
		return u.onFrame(NewFrame(FrameVoice, 10, u.frame))
	default:
		return ErrInvalidG729Frame
	}
}
