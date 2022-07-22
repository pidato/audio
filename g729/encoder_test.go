package g729

import (
	"fmt"
	"github.com/pidato/audio/g711"
	"github.com/pidato/audio/util"
	"github.com/pidato/audio/vad"
	"os"
	"testing"
)

func pcmFrame(seq, samples int, b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	idx := seq * (samples * 2)
	if idx+(samples*2) > len(b) {
		return nil
	}
	return b[idx : idx+(samples*2)]
}

func TestEncoder(t *testing.T) {
	pcm, err := os.ReadFile("testdata/8000.pcm")
	if err != nil {
		t.Fatal(err)
	}
	doEncode(t, false, pcm)
	doEncode(t, true, pcm)
}

func doEncode(t *testing.T, enableVAD bool, pcm []byte) {
	encoder := NewEncoder(enableVAD)
	defer encoder.Close()

	name := "testdata/novad.g729"
	if enableVAD {
		name = "testdata/vad.g729"
	}
	out, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = out.Close()
	}()
	bitstream := make([]byte, 10)
	var (
		countZero = 0
		countTwo  = 0
		countTen  = 0
	)
LOOP:
	for i := 0; ; i++ {
		frame := pcmFrame(i, 80, pcm)
		if len(frame) == 0 {
			break LOOP
		}
		n, err := encoder.Encode(util.CastBytesToInt16(frame), bitstream)
		if err != nil {
			t.Fatal(err)
		}
		switch n {
		case 0:
			countZero++
		case 2:
			fmt.Println("Comfort Noise: ", bitstream[0], bitstream[1])
			countTwo++
		case 10:
			countTen++
		}
		_ = n
		//fmt.Println("Seq", i, "BitstreamLength", n)
		_, err = out.Write(bitstream)
		if err != nil {
			t.Fatal(err)
		}
	}

	fmt.Println(name, "0 =", countZero, " 2 =", countTwo, " 10 =", countTen)
}

func TestEncodeVAD(t *testing.T) {
	pcm, err := os.ReadFile("testdata/8000.pcm")
	if err != nil {
		t.Fatal(err)
	}
	encoder := NewEncoder(true)
	defer encoder.Close()

	v, _ := vad.New(8000, vad.LowBitrate)

	bitstream := make([]byte, 10)
	var (
		countZero = 0
		countTwo  = 0
		countTen  = 0

		gVADState = vad.Noise
		vadState  = vad.Noise
	)
LOOP:
	for i := 0; ; i++ {
		frame := pcmFrame(i, 80, pcm)
		if len(frame) == 0 {
			break LOOP
		}
		nextVAD := v.Process(util.CastBytesToInt16(frame))
		if nextVAD != vadState {
			fmt.Println("WebRTC", "  @", i, "  ", vadState.String(), " -> ", nextVAD.String())
			vadState = nextVAD
		}
		n, err := encoder.Encode(util.CastBytesToInt16(frame), bitstream)
		if err != nil {
			t.Fatal(err)
		}
		switch n {
		case 0:
			countZero++
		case 2:
			countTwo++
		case 10:
			countTen++
		}
		nextGVAD := bitLengthToVAD(n)
		if nextGVAD != gVADState {
			fmt.Println("  G729", "  @", i, "  ", gVADState.String(), " -> ", nextGVAD.String())
			gVADState = nextGVAD
		}
	}
}

func bitLengthToVAD(l uint8) vad.State {
	switch l {
	case 0, 2:
		return vad.Noise
	default:
		return vad.Voice
	}
}

func BenchmarkEncoder(b *testing.B) {
	pcm, err := os.ReadFile("testdata/8000.pcm")
	if err != nil {
		b.Fatal(err)
	}

	b.Run("No VAD", func(b *testing.B) {
		b.SetBytes(int64(len(pcm)))
		bitstream := make([]byte, 10)
		for i := 0; i < b.N; i++ {
			encoder := NewEncoder(false)
		LOOP:
			for i := 0; ; i++ {
				frame := pcmFrame(i, 80, pcm)
				if len(frame) == 0 {
					break LOOP
				}
				_, err := encoder.Encode(util.CastBytesToInt16(frame), bitstream)
				if err != nil {
					b.Fatal(err)
				}
			}
			_ = encoder.Close()
		}
	})

	b.Run("VAD", func(b *testing.B) {
		b.SetBytes(int64(len(pcm)))
		bitstream := make([]byte, 10)
		for i := 0; i < b.N; i++ {
			encoder := NewEncoder(true)
		LOOP:
			for i := 0; ; i++ {
				frame := pcmFrame(i, 80, pcm)
				if len(frame) == 0 {
					break LOOP
				}
				_, err := encoder.Encode(util.CastBytesToInt16(frame), bitstream)
				if err != nil {
					b.Fatal(err)
				}
			}
			_ = encoder.Close()
		}
	})

	b.Run("VAD Recycle", func(b *testing.B) {
		b.SetBytes(int64(len(pcm)))
		bitstream := make([]byte, 10)
		encoder := NewEncoder(true)
		for i := 0; i < b.N; i++ {
		LOOP:
			for i := 0; ; i++ {
				frame := pcmFrame(i, 80, pcm)
				if len(frame) == 0 {
					break LOOP
				}
				_, err := encoder.Encode(util.CastBytesToInt16(frame), bitstream)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
		_ = encoder.Close()
	})

	b.Run("ULAW", func(b *testing.B) {
		b.SetBytes(int64(len(pcm)))
		ulaw := make([]byte, 80)
		for i := 0; i < b.N; i++ {
		LOOP:
			for i := 0; ; i++ {
				frame := pcmFrame(i, 80, pcm)
				if len(frame) == 0 {
					break LOOP
				}
				g711.EncodeUlawTo(ulaw, util.CastBytesToInt16(frame))
			}
		}
	})

	b.Run("ULAW and VAD", func(b *testing.B) {
		b.SetBytes(int64(len(pcm)))
		ulaw := make([]byte, 80)
		v, _ := vad.New(8000, vad.LowBitrate)
		for i := 0; i < b.N; i++ {
		LOOP:
			for i := 0; ; i++ {
				frame := pcmFrame(i, 80, pcm)
				if len(frame) == 0 {
					break LOOP
				}
				_ = v.Process(util.CastBytesToInt16(frame))
				g711.EncodeUlawTo(ulaw, util.CastBytesToInt16(frame))
			}
		}
	})
}
