package pcm

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestBuffer_Write(t *testing.T) {
	reader, err := OpenWavFile("testdata/recording.wav", Ptime20)
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := NewBuffer(16000, 10, 100)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		defer reader.Close()
		count := 0
		for {
			frame, err := reader.ReadFrame()
			if err != nil {
				if len(frame) > 0 {
					_ = buffer.Write(frame)
					count++
				}
				_ = buffer.WriteFinal()
				return
			}

			count++
			err = buffer.WriteBlocking(frame)
			if err != nil {
				if err != os.ErrClosed {
					t.Fatal(err)
				}
				return
			}

			// Simulate a pause.
			//time.Sleep(time.Microsecond * 100)
		}
	}()

	count := 0
	for {
		frame, err := buffer.ReadFrame()
		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF")
				break
			}
			t.Fatal(err)
		}

		_ = frame
		fmt.Println(count)
		count++
	}
	_ = buffer.Close()
	fmt.Println(buffer.MaxDivergence())
}
