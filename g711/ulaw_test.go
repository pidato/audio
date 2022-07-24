package g711

import (
	"testing"
)

func TestEncodeUlaw(t *testing.T) {
	if EncodeULAWFrame(0) != 255 {
		t.Error("omg")
	}
	if EncodeULAWFrame(-100) != 114 {
		t.Error("omg")
	}
}

func TestDecodeUlaw(t *testing.T) {
	for n := 0; n <= 255; n++ {
		if DecodeUlawFrame(uint8(n)) != ulaw2lpcm[n] {
			t.Error("omg")
			return
		}
	}
}

func TestDecodeUlawEncodeDecode(t *testing.T) {
	for n := 0; n <= 255; n++ {
		if DecodeUlawFrame(EncodeULAWFrame(ulaw2lpcm[n])) != ulaw2lpcm[n] {
			t.Error("omg")
			return
		}
	}
}
