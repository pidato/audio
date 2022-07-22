//go:build !amd64 && (tinygo.wasm || 386 || amd64 || amd64p32 || arm || arm64 || loong64 || mips64le || mips64p32 || mips64p32le || mipsle || ppc64le || riscv || riscv64 || wasm)

package pcm

import (
	"github.com/pidato/audio/g711"
	"math/bits"
	"unsafe"
)

// Mixes together two audio frames containing 160 samples. Uses saturation
// adding so you don't hear clipping if ppl talk loud. Uses 128-bit SIMD
// instructions so we can add eight numbers at the same time.
func Mix160(dst, src *int16) {
	var (
		d = unsafe.Pointer(dst)
		s = unsafe.Pointer(src)
	)
	for i := 0; i < 160; i += 2 {
		var (
			av = bits.Reverse16(*(*uint16)(unsafe.Add(d, i)))
			bv = bits.Reverse16(*(*uint16)(unsafe.Add(s, i)))
			a  = int32(*(*int16)(unsafe.Pointer(&av)))
			b  = int32(*(*int16)(unsafe.Pointer(&bv)))
			m  int32
		)
		// Make both samples unsigned (0..65535)
		a += 32768
		b += 32768
		// Pick the equation
		if a < 32768 || b < 32768 {
			// Viktor's first equation when both sources are "quiet"
			// (i.e. less than middle of the dynamic range)
			m = a * b / 32768
		} else {
			// Viktor's second equation when one or both sources are loud
			m = 2*(a+b) - (a*b)/32768 - 65536
		}

		// Output is unsigned (0..65536) so convert back to signed (-32768..32767)
		if m == 65536 {
			m = 65535
		}
		m -= 32768
		*(*int16)(unsafe.Add(d, i)) = int16(m)
	}
}

// Compresses a PCM audio sample into a G.711 μ-Law sample. The BSR instruction
// is what makes this code fast.
func LinearToUlaw(linear int64) (ulaw int64) {
	return int64(g711.EncodeUlawFrame(int16(linear)))
}

// Turns a μ-Law byte back into an audio sample.
func UlawToLinear(ulaw int64) (linear int64) {
	return int64(g711.DecodeUlawFrame(uint8(ulaw)))
}
