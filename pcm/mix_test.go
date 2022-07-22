package pcm

import (
	"testing"
)

func TestMix(t *testing.T) {
	var x, y [160]int16
	for n := 0; n < 160; n++ {
		x[n] = int16(n)
		y[n] = int16(666)
	}
	Mix(x[0:len(x)], y[0:len(y)])
	for n := 0; n < len(x); n++ {
		want := int16(n + 666)
		if x[n] != want {
			t.Errorf("x[%v] = %v (wanted: %v)", n, x[n], want)
			return
		}
		if y[n] != int16(666) {
			t.Errorf("side effect y[%v] = %v", n, y[n])
			return
		}
	}
}
