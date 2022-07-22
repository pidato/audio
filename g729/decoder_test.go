package g729

import (
	"github.com/pidato/audio/g711"
	"github.com/pidato/audio/util"
	"os"
	"testing"
)

func doDecode(t *testing.T, name string, outName string) {
	encoded, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.OpenFile(outName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	decoder := NewDecoder()

	pcm := make([]int16, 80)
LOOP:
	for i := 0; ; i++ {
		if i*10+10 > len(encoded) {
			break LOOP
		}
		encodedFrame := encoded[i*10 : i*10+10]
		err = decoder.Decode(encodedFrame, pcm)
		if err != nil {
			t.Fatal(err)
		}
		//if isVoice != decoder.IsVoice() {
		//	if isVoice {
		//		fmt.Println("voiced stopped", i)
		//		isVoice = false
		//	} else {
		//		fmt.Println("voice started", i)
		//		isVoice = true
		//	}
		//}
		_, err = out.Write(g711.EncodeUlaw(util.CastInt16ToBytes(pcm)))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDecoder(t *testing.T) {
	pcm, err := os.ReadFile("testdata/8000.pcm")
	if err != nil {
		t.Fatal(err)
	}
	doEncode(t, false, pcm)
	doEncode(t, true, pcm)
	doDecode(t, "testdata/novad.g729", "testdata/novad.ulaw")
	doDecode(t, "testdata/vad.g729", "testdata/vad.ulaw")
	doDecode(t, "testdata/vad1.g729", "testdata/vad1.ulaw")
}

func BenchmarkDecoder(b *testing.B) {
	b.Run("VAD", func(b *testing.B) {
		encoded, err := os.ReadFile("testdata/vad1.g729")
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(encoded) / 10 * 160))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder()

			pcm := make([]int16, 80)
		LOOP:
			for i := 0; ; i++ {
				if i*10+10 > len(encoded) {
					break LOOP
				}
				encodedFrame := encoded[i*10 : i*10+10]
				err = decoder.Decode(encodedFrame, pcm)
				if err != nil {
					b.Fatal(err)
				}
			}
			_ = decoder.Close()
		}
	})

	b.Run("No VAD", func(b *testing.B) {
		encoded, err := os.ReadFile("testdata/novad.g729")
		if err != nil {
			b.Fatal(err)
		}
		b.SetBytes(int64(len(encoded) / 10 * 160))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			decoder := NewDecoder()

			pcm := make([]int16, 80)
		LOOP:
			for i := 0; ; i++ {
				if i*10+10 > len(encoded) {
					break LOOP
				}
				encodedFrame := encoded[i*10 : i*10+10]
				err = decoder.Decode(encodedFrame, pcm)
				if err != nil {
					b.Fatal(err)
				}
			}
			_ = decoder.Close()
		}
	})
}
