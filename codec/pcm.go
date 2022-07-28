package codec

type pcm8 struct{}

func (pcm8) PayloadType() uint8 {
	return 16
}

func (pcm8) Name() string {
	return "L16/8000"
}

func (pcm8) SampleRate() int32 {
	return 8000
}

type pcm16 struct{}

func (pcm16) PayloadType() uint8 {
	return 16
}

func (pcm16) Name() string {
	return "L16/16000"
}

func (pcm16) SampleRate() int32 {
	return 16000
}
