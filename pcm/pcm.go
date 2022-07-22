package pcm

const (
	Ptime10 = 10
	Ptime20 = 20
)

func I16toF32(i int16) float32 {
	if i == 0 {
		return 0
	}
	f := (float32(i)) / float32(32768)
	if f > 1 {
		f = 1.0
	}
	if f < -1 {
		f = -1
	}
	return f
}

func F32ToI16(f float32) int16 {
	f = f * 32768
	if f > 32767 {
		f = 32767
	}
	if f < -32768 {
		f = -32768
	}
	return int16(f)
}

func CopyI16(dst []int16, src []float32) []int16 {
	if len(dst) < len(src) {
		dst = make([]int16, len(src))
	}
	for i := 0; i < len(src); i++ {
		dst[i] = F32ToI16(src[i])
	}
	return dst
}

func CopyF32(dst []float32, src []int16) []float32 {
	if len(dst) < len(src) {
		dst = make([]float32, len(src))
	}
	for i := 0; i < len(src); i++ {
		dst[i] = I16toF32(src[i])
	}
	return dst
}
