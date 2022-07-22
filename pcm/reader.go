package pcm

import (
	"io"
	"time"
)

// 16bit PCM reader.
type Reader interface {
	io.Closer

	Elapsed() time.Duration

	// Clock speed in hertz. (i.e. 16000 for 16Khz)
	SampleRate() int

	// Number of 16bit integers per frame.
	FrameSize() int

	// Payload duration.
	Ptime() time.Duration

	// Release buffer to allow it to be recycled.
	Release(p []int16)

	// Allocate a new Payload.
	Alloc() []int16

	// ReadFrame the next Payload.
	ReadFrame() ([]int16, error)
}

type RawReader interface {
	Read(p []int16) (n int, err error)
}
