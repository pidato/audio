package transcode

import (
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/g729"
	"github.com/pidato/audio/pool"
	"github.com/pidato/audio/vad"
	"io"
	"time"
)

var _ Transcoder = &ulawG729{}

type ulawG729 struct {
	pcm      []int16
	in       []byte
	encoded  []byte
	stats    Stats
	enc      *g729.Encoder
	vad      *vad.VAD
	vadState vad.State
	vadAlgo  VADAlgo
	onFrame  OnFrame
}

func newUlawG729(algo VADAlgo, onFrame OnFrame) (*ulawG729, error) {
	if onFrame == nil {
		onFrame = func(frame Frame) error {
			return nil
		}
	}
	u := &ulawG729{
		stats: Stats{
			ptime: 10,
		},
		pcm:     pool.I16Get(80),
		encoded: pool.U8Get(10),
		onFrame: onFrame,
		vadAlgo: algo,
	}
	switch algo {
	case WebRTCQuality:
		u.vad, _ = vad.New(8000, vad.Quality)
	case WebRTCLowBitrate:
		u.vad, _ = vad.New(8000, vad.LowBitrate)
	case WebRTCAggressive:
		u.vad, _ = vad.New(8000, vad.Aggressive)
	case WebRTCVeryAggressive:
		u.vad, _ = vad.New(8000, vad.VeryAggressive)
	}
	u.enc = g729.NewEncoder(true)
	return u, nil
}

func (u *ulawG729) From() codec.Codec { return codec.ULAW }

func (u *ulawG729) To() codec.Codec { return codec.G729 }

func (u *ulawG729) Algo() VADAlgo {
	return u.vadAlgo
}

func (u *ulawG729) VAD() vad.State {
	return u.vadState
}

func (u *ulawG729) Ptime() codec.Ptime {
	return 10
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
	if u.encoded != nil {
		pool.U8Release(u.encoded)
		u.encoded = nil
	}
	return nil
}

func (u *ulawG729) Dropped(ptime codec.Ptime) error {
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

func (u *ulawG729) Transcode(kind FrameType, b []byte) error {
	if kind == FrameDropped {
		u.stats.addFrames(1)
		u.stats.addDropped(1)
		return u.onFrame(NewFrame(FrameDropped, 10, nil))
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
	u.stats.addFrames(1)
	pcm := u.pcm
	codec.DecodeULAW(pcm, b)
	length, err := u.enc.Encode(pcm, u.encoded)
	if err != nil {
		return err
	}
	if u.vad != nil {
		u.vadState = u.vad.Process(pcm)
		if u.vadState == vad.Voice {
			u.stats.addVoice(1)
			return u.onFrame(NewFrame(FrameVoice, 10, u.encoded[0:length]))
		} else {
			u.stats.addNoise(1)
			return u.onFrame(NewFrame(FrameNoise, 10, u.encoded[0:length]))
		}
	} else {
		switch length {
		case 0:
			u.stats.addNoise(1)
			return u.onFrame(NewFrame(FrameNoise, 10, u.encoded[0:0]))
		case 2:
			u.stats.addComfort(1)
			return u.onFrame(NewFrame(FrameNoise, 10, u.encoded[0:2]))
		default:
			u.stats.addVoice(1)
			return u.onFrame(NewFrame(FrameVoice, 10, u.encoded))
		}
	}
}
