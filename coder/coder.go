package coder

import (
	"errors"
	"github.com/pidato/audio/codec"
	"github.com/pidato/audio/vad"
	"io"
	"sync/atomic"
	"time"
)

var (
	ErrPartialFrame     = errors.New("partial frame")
	ErrInvalidG729Frame = errors.New("invalid g729 frame")
	ErrNotSupported     = errors.New("not supported")
	ErrInvalidPtime     = errors.New("invalid ptime")
)

type Coder interface {
	io.Closer

	From() codec.Codec

	To() codec.Codec

	Algo() VADAlgo

	VAD() vad.State

	Ptime() codec.Ptime

	Stats() *Stats

	Duration() time.Duration

	Dropped(ptime codec.Ptime) error

	Transcode(kind FrameType, b []byte) error
}

func New(
	from codec.Codec,
	to codec.Codec,
	algo VADAlgo,
	ptime codec.Ptime,
	onFrame OnFrame,
) (Coder, error) {
	if from == codec.ULAW {
		if to == codec.G729 {
			return newULAWG729(algo, onFrame)
		}
		if to == codec.ULAW {
			return newULAW2ULAW(algo, ptime, onFrame)
		}
		return nil, ErrNotSupported
	}
	if from == codec.G729 {
		if to == codec.ULAW {
			return newG729ulaw(algo, onFrame)
		}
		if to == codec.G729 {
			return newG729G729(algo, onFrame)
		}
		return nil, ErrNotSupported
	}
	return nil, ErrNotSupported
}

//func Forward(c codec.Codec, algo VADAlgo) Coder {
//	return &forward{
//		c:     c,
//		algo:  algo,
//		vad:   vad.Voice,
//		stats: Stats{},
//	}
//}
//
//type forward struct {
//	c     codec.Codec
//	algo  VADAlgo
//	vad   vad.State
//	stats Stats
//}
//
//func (f *forward) Close() error { return nil }
//
//func (f *forward) From() codec.Codec { return f.c }
//
//func (f *forward) To() codec.Codec { return f.c }
//
//func (f *forward) Algo() VADAlgo { return f.algo }
//
//func (f *forward) VAD() vad.State { return f.vad }
//
//func (f *forward) Ptime() codec.Ptime { return 10 }
//
//func (f *forward) Stats() *Stats { return &f.stats }
//
//func (f *forward) Duration() time.Duration { return f.stats.Duration() }
//
//func (f *forward) Dropped(ptime codec.Ptime) error { return nil }
//
//func (f *forward) Transcode(kind FrameType, b []byte) error {
//	switch kind {
//	case FrameVoice:
//
//	}
//	return nil
//}

type Frame struct {
	Kind     FrameType
	Duration codec.Ptime
	Payload  []byte
}

func NewFrame(kind FrameType, duration codec.Ptime, payload []byte) Frame {
	return Frame{Kind: kind, Duration: duration, Payload: payload}
}

type OnFrame func(frame Frame) error

type VADAlgo byte

const (
	NoVAD                VADAlgo = 0
	WebRTCQuality        VADAlgo = 1
	WebRTCLowBitrate     VADAlgo = 2
	WebRTCAggressive     VADAlgo = 3
	WebRTCVeryAggressive VADAlgo = 4
	G729VAD              VADAlgo = 5
	OpusVAD              VADAlgo = 6
)

type FrameType byte

const (
	FrameNoise   FrameType = 0
	FrameVoice   FrameType = 1
	FrameDropped FrameType = 2
	FrameEOF     FrameType = 3
)

const (
	_10ms = time.Millisecond * 10
)

type Stats struct {
	ptime   codec.Ptime
	frames  int64
	dropped int64
	comfort int64
	voice   int64
	noise   int64
}

func (s *Stats) addFrames(count int64) {
	atomic.AddInt64(&s.frames, count)
}
func (s *Stats) addVoice(count int64) {
	atomic.AddInt64(&s.voice, count)
}
func (s *Stats) addNoise(count int64) {
	atomic.AddInt64(&s.noise, count)
}
func (s *Stats) addDropped(count int64) {
	atomic.AddInt64(&s.dropped, count)
}
func (s *Stats) addComfort(count int64) {
	atomic.AddInt64(&s.comfort, count)
}

func (s *Stats) Ptime() codec.Ptime {
	return s.ptime
}
func (s *Stats) Frames() int64 {
	return atomic.LoadInt64(&s.frames)
}
func (s *Stats) Voice() int64 {
	return atomic.LoadInt64(&s.voice)
}
func (s *Stats) Noise() int64 {
	return atomic.LoadInt64(&s.noise)
}
func (s *Stats) Comfort() int64 {
	return atomic.LoadInt64(&s.comfort)
}
func (s *Stats) Dropped() int64 {
	return atomic.LoadInt64(&s.dropped)
}

func (s *Stats) Duration() time.Duration {
	return time.Duration(s.frames) * time.Duration(s.ptime) * time.Millisecond
}

func (s *Stats) Clone() Stats {
	return *s
}
