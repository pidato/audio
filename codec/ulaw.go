package codec

type ulaw struct{}

func (ulaw) BytesPerFrame(ptime int) int {
	return ptime * 8
}

func (ulaw) PayloadType() uint8 {
	return 0
}

func (ulaw) Name() string {
	return "PCMU/8000"
}

func (ulaw) SampleRate() int32 {
	return 8000
}
