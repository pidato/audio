package opus

import (
	"fmt"
	"github.com/pidato/audio/vad"
	"io"
	"os"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

func TestReader(t *testing.T) {
	//reader, err := OpenFile("testdata/speech_8.opus")
	reader, err := OpenFile("../opus/testdata/1.opus")
	if err != nil {
		t.Fatal(err)
	}

	sampleRate := int(reader.SampleRate())

	if sampleRate == 44100 {
		sampleRate = 48000
	}

	dec, err := NewDecoder(sampleRate, int(reader.ChannelCount()))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Channels: %d\n", reader.ChannelCount())
	pcm := make([]int16, 20000)

	pcmFile, err := os.OpenFile("../opus/testdata/1.pcm", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		t.Fatal(err)
	}

	multiple := xMAX_BITRATE / sampleRate

	preSkip := int(reader.PreSkip()) / multiple
	totalSamples := 0
	var duration time.Duration

	sampleDuration := time.Second / 16000

	fmt.Println(reader.PreSkip())
	fmt.Println(preSkip)

	v, _ := vad.New(16000, vad.Aggressive)

	granulePosition := 0
	_ = granulePosition

	for {
		// Reads the next packet.
		packet, err := reader.ReadPacket()

		if len(packet) > 0 {
			n, err := dec.Decode(packet, pcm)
			if err != nil {
				t.Fatal(err)
			}

			dur, err := dec.LastPacketDuration()
			fmt.Println("LastPacketDuration", dur)

			pcm = pcm[:n]
			granulePosition += (len(pcm) * multiple)

			pcmBytes := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
				Data: (uintptr)(unsafe.Pointer(&pcm[0])),
				Len:  len(pcm) * 2,
				Cap:  len(pcm) * 2,
			}))

			v.Process(pcm)

			if preSkip > 0 {

			}

			totalSamples += len(pcm)
			duration += sampleDuration * time.Duration(len(pcm))

			fmt.Printf("Payload:\n")
			fmt.Printf("\tOpus: %d\n", len(packet))
			fmt.Printf("\tPage Granule Sequence: %d\n", reader.GranulePos())
			fmt.Printf("\tGranule Sequence: %d\n", granulePosition)
			fmt.Printf("\tPCM: %d\n", len(pcm))

			_, _ = pcmFile.Write(pcmBytes)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}

	fmt.Printf("Total Samples: %d\n", totalSamples)
	fmt.Printf("Total Duration: %v\n", duration)
	fmt.Printf("Total Duration - PreSkip: %v\n", duration-(sampleDuration*time.Duration(int(reader.PreSkip())/multiple)))

	err = pcmFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
}
