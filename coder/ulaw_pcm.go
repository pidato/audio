package coder

import (
	"errors"
	"fmt"
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/g711"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/util"
	"github.com/pidato/audio/vad"
	"io"
	"time"
)

var _ Coder = &ulawpcm{}

type ulawpcm struct {
	to             codec.Codec
	pcm            []int16
	in             []byte
	upsampled      []int16
	upsampledState [8]int32
	frameSize      int
	stats          Stats
	vad            *vad.VAD
	algo           VADAlgo
	vadState       vad.State
	onFrame        OnFrame
}

func newULAWPCM(to codec.Codec, algo VADAlgo, ptime codec.Ptime, onFrame OnFrame) (*ulawpcm, error) {
	if to == nil {
		return nil, errors.New("to codec is nil")
	}
	if onFrame == nil {
		onFrame = func(kind Frame) error {
			return nil
		}
	}
	ptime.Validate()
	u := &ulawpcm{
		to:        to,
		onFrame:   onFrame,
		frameSize: int(ptime) * 8,
		algo:      algo,
		stats: Stats{
			ptime: ptime,
		},
	}
	switch to.SampleRate() {
	case 8000:
	case 16000:
		u.upsampled = pool.I16Get(u.frameSize * 2 * 2)
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
		u.vad, _ = vad.New(8000, vad.LowBitrate)
	}
	if u.vad != nil {
		u.pcm = pool.I16Get(u.frameSize)
	} else {
		u.vadState = vad.Voice
	}
	return u, nil
}

func (u *ulawpcm) From() codec.Codec { return codec.ULAW }

func (u *ulawpcm) To() codec.Codec { return u.to }

func (u *ulawpcm) Algo() VADAlgo {
	return u.algo
}

func (u *ulawpcm) VAD() vad.State {
	return u.vadState
}

func (u *ulawpcm) Ptime() codec.Ptime {
	return u.stats.ptime
}

func (u *ulawpcm) Stats() *Stats {
	return &u.stats
}

func (u *ulawpcm) Duration() time.Duration {
	return u.Duration()
}

func (u *ulawpcm) Close() error {
	if u.vad != nil {
		_ = u.vad.Close()
		u.vad = nil
	}
	if u.pcm != nil {
		pool.I16Release(u.pcm)
		u.pcm = nil
	}
	if u.in != nil {
		pool.U8Release(u.in)
		u.in = nil
	}
	if u.upsampled != nil {
		pool.I16Release(u.upsampled)
		u.upsampled = nil
	}
	return nil
}

func (u *ulawpcm) push(b []byte) error {
	if u.vad != nil {
		g711.DecodeULAW(u.pcm, b)
		u.vadState = u.vad.Process(u.pcm)

		frame := u.pcm
		if u.upsampled != nil {
			if err := vad.UpSampleBy2(u.pcm, u.upsampled, &u.upsampledState); err != nil {
				return err
			}
			frame = u.upsampled
		}

		if u.vadState == vad.Voice {
			return u.onFrame(NewFrame(FrameVoice, u.stats.ptime, util.CastInt16ToBytes(frame)))
		} else {
			return u.onFrame(NewFrame(FrameNoise, u.stats.ptime, util.CastInt16ToBytes(frame)))
		}
	} else {
		frame := u.pcm
		if u.upsampled != nil {
			if err := vad.UpSampleBy2(u.pcm, u.upsampled, &u.upsampledState); err != nil {
				return err
			}
			frame = u.upsampled
		}
		return u.onFrame(NewFrame(FrameVoice, u.stats.ptime, util.CastInt16ToBytes(frame)))
	}
}

func (u *ulawpcm) Dropped(ptime codec.Ptime) error {
	if ptime == 0 {
		return nil
	}
	u.stats.addDropped(int64(ptime))
	return u.onFrame(NewFrame(FrameDropped, ptime, nil))
}

func (u *ulawpcm) Transcode(kind FrameType, b []byte) error {
	switch kind {
	case FrameDropped:
		return u.onFrame(NewFrame(FrameDropped, u.stats.ptime, b))
	case FrameNoise:
		return u.onFrame(NewFrame(FrameNoise, u.stats.ptime, b))
	}
	if len(b) == 0 {
		return io.ErrShortBuffer
	}
	if len(u.in) == 0 {
		if len(b) == u.frameSize {
			return u.push(b)
		}
		if len(b) < u.frameSize {
			if u.in == nil {
				u.in = pool.U8Get(u.frameSize)
			}
			u.in = append(u.in, b...)
			return nil
		}
		err := u.push(b[0:u.frameSize])
		if err != nil {
			return err
		}
		b = b[u.frameSize:]

		return nil
	}

	l := len(u.in) + len(b)
	if l < u.frameSize {
		u.in = append(u.in, b...)
		return nil
	}
	if l == u.frameSize {
		u.in = append(u.in, b...)
		err := u.push(u.in)
		u.in = u.in[:0]
		return err
	}
	u.in = append(u.in, b[0:u.frameSize-len(u.in)]...)
	b = b[u.frameSize-len(u.in):]
	err := u.push(u.in)
	u.in = u.in[:0]
	if err != nil {
		return err
	}

	for {
		if len(b) < u.frameSize {
			if len(b) > 0 {
				u.in = append(u.in, b...)
			}
			return nil
		}

		err = u.push(b[0:u.frameSize])
		if err != nil {
			return err
		}
		b = b[u.frameSize:]
	}
}
