package coder

import (
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/g711"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/vad"
	"io"
	"time"
)

var _ Coder = &ulaw2ulaw{}

type ulaw2ulaw struct {
	pcm       []int16
	in        []byte
	frameSize int
	stats     Stats
	vad       *vad.VAD
	algo      VADAlgo
	vadState  vad.State
	onFrame   OnFrame
}

func newULAW2ULAW(algo VADAlgo, ptime codec.Ptime, onFrame OnFrame) (*ulaw2ulaw, error) {
	if ptime < 0 {
		ptime = 10
	}
	if ptime%10 != 0 {
		return nil, ErrInvalidPtime
	}
	if onFrame == nil {
		onFrame = func(kind Frame) error {
			return nil
		}
	}
	u := &ulaw2ulaw{
		onFrame:   onFrame,
		frameSize: int(ptime) * 8,
		algo:      algo,
		stats: Stats{
			ptime: ptime,
		},
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

func (u *ulaw2ulaw) From() codec.Codec { return codec.ULAW }

func (u *ulaw2ulaw) To() codec.Codec { return codec.ULAW }

func (u *ulaw2ulaw) Algo() VADAlgo {
	return u.algo
}

func (u *ulaw2ulaw) VAD() vad.State {
	return u.vadState
}

func (u *ulaw2ulaw) Ptime() codec.Ptime {
	return u.stats.ptime
}

func (u *ulaw2ulaw) Stats() *Stats {
	return &u.stats
}

func (u *ulaw2ulaw) Duration() time.Duration {
	return u.Duration()
}

func (u *ulaw2ulaw) Close() error {
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
	return nil
}

func (u *ulaw2ulaw) push(b []byte) error {
	if u.vad != nil {
		g711.DecodeULAW(u.pcm, b)
		u.vadState = u.vad.Process(u.pcm)
		if u.vadState == vad.Voice {
			return u.onFrame(NewFrame(FrameVoice, u.stats.ptime, b))
		} else {
			return u.onFrame(NewFrame(FrameNoise, u.stats.ptime, b))
		}
	} else {
		return u.onFrame(NewFrame(FrameVoice, u.stats.ptime, b))
	}
}

func (u *ulaw2ulaw) Dropped(ptime codec.Ptime) error {
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

func (u *ulaw2ulaw) Transcode(kind FrameType, b []byte) error {
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
