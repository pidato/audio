//go:build arm64be || armbe || mips || mips64 || ppc || ppc64 || s390 || s390x || sparc || sparc64

package pcm

import (
	"github.com/pidato/audio/util"
	"unsafe"
)

func Mix(dst, src []int16) {
	size := len(dst)
	if len(src) < size {
		size = len(src)
	}
	var (
		d = util.I16SliceToPointer(dst)
		s = util.I16SliceToPointer(src)
	)
	for i := 0; i < size; i += 2 {
		var (
			a = int32(*(*int16)(unsafe.Add(d, i)))
			b = int32(*(*int16)(unsafe.Add(s, i)))
			m int32
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

func MixInto(dst, srcA, srcB []int16) {
	size := len(srcA)
	if len(srcB) < size {
		size = len(srcB)
	}
	var (
		sa = util.I16SliceToPointer(srcA)
		sb = util.I16SliceToPointer(srcB)
		d  = util.I16SliceToPointer(dst)
	)
	for i := 0; i < size; i += 2 {
		var (
			a = int32(*(*int16)(unsafe.Add(sa, i)))
			b = int32(*(*int16)(unsafe.Add(sb, i)))
			m int32
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
