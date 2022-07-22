package pcm

import (
	"fmt"
	"io"
	"testing"
)

func TestOpenWav_Read(t *testing.T) {
	reader, err := OpenWavFile("testdata/recording.wav", Ptime10)
	if err != nil {
		t.Fatal(err)
	}

	buffer := reader.Alloc()

	count := 0
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			t.Fatal(err)
			break
		}
		if n == 0 {
			break
		}
		if len(buffer) != n {
			buffer = buffer[:n]
			fmt.Printf("Partial Payload: %d\n", count)
			break
		}

		fmt.Printf("Payload: %d\n", count)
		count++
	}

	fmt.Println("EOF")
}

func TestOpenWav_PoolAlloc(t *testing.T) {
	reader, err := OpenWavFile("testdata/recording.wav", Ptime20)
	if err != nil {
		t.Fatal(err)
	}

	frameSize := reader.FrameSize()

	count := 0
	for {
		buffer, err := reader.ReadFrame()
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if len(buffer) == 0 {
			break
		}
		if len(buffer) != frameSize {
			fmt.Printf("Partial Payload: %d\n", count)
			break
		}

		fmt.Printf("Payload: %d\n", count)
		count++

		reader.Release(buffer)
	}

	fmt.Println("EOF")
}
