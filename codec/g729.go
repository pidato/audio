package codec

type g729codec struct{}

func (g729codec) PayloadType() uint8 {
	return 18
}

func (g729codec) Name() string {
	return "G729/8000"
}

func (g729codec) SampleRate() int32 {
	return 8000
}
