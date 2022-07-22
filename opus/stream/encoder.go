package stream

import (
	"github.com/pidato/audio/opus"
	"github.com/pidato/audio/pcm"
)

type Encoder struct {
	enc *opus.Encoder
	pcm *pcm.Buffer
}

func NewEncoder(sampleRate, ptime, maxFrames int, app opus.Application) (*Encoder, error) {
	buf, err := pcm.NewBuffer(sampleRate, ptime, maxFrames)
	if err != nil {
		return nil, err
	}

	enc, err := opus.NewEncoder(sampleRate, 1, app)

	return &Encoder{
		enc: enc,
		pcm: buf,
	}, nil
}

func (e *Encoder) Write(frame []int16) (n int, err error) {
	return
}
