package pcm

import (
	"io"
	"sync"
	"time"
)

const (
	DefaultLimit = 0
)

// ReplayReader keeps a buffer of recently read frames. This comes in handy for
// using Voice-Activity Detection (VAD) to determine utterance windows. Being able
// to look back "x" number of frames to catch the true beginning of the utterance.
//
// Wraps the backing reader which it utilizes for Buffer management. Buffers are
// released internally once the buffer is removed from the internal history buffer.
// When "Close" is called, the entire history buffer is released to the backing reader.
type ReplayReader struct {
	reader Reader
	count  int
	buffer [][]int16
	closed bool
	mu     sync.Mutex
}

func NewReplayReader(reader Reader, duration time.Duration) *ReplayReader {
	if duration <= time.Duration(0) {
		duration = time.Millisecond * 60
	}
	ptime := reader.Ptime()
	if ptime < Ptime10 {
		ptime = Ptime10
	}
	frames := int(duration / ptime)
	if frames <= 0 {
		frames = 1
	}

	return &ReplayReader{
		reader: reader,
		count:  0,
		buffer: make([][]int16, frames),
		closed: false,
	}
}

func (r *ReplayReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return io.ErrClosedPipe
	}
	for i := 0; i < len(r.buffer); i++ {
		buf := r.buffer[i]
		if buf != nil {
			r.buffer[i] = nil
			r.reader.Release(buf)
		}
	}
	return r.reader.Close()
}

func (w *ReplayReader) Elapsed() time.Duration {
	return w.reader.Elapsed()
}

func (r *ReplayReader) Replay(limit int, fn func(p []int16)) (n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0
	}

	if fn == nil {
		return 0
	}
	if limit < 0 {
		return 0
	}
	if limit == 0 {
		limit = len(r.buffer)
	} else if limit > len(r.buffer) {
		limit = len(r.buffer)
	}

	end := r.count - 1
	start := end - limit

	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	n = 0
	for i := start; i < end; i++ {
		idx := i % len(r.buffer)
		buf := r.buffer[idx]
		if len(buf) > 0 {
			fn(buf)
		}
		n++
	}
	return n
}

// Clock speed in hertz. (i.e. 16000 for 16Khz)
func (r *ReplayReader) SampleRate() int {
	return r.reader.SampleRate()
}

// Number of 16bit integers per frame.
func (r *ReplayReader) FrameSize() int {
	return r.reader.FrameSize()
}

// Payload duration.
func (r *ReplayReader) Ptime() time.Duration {
	return r.reader.Ptime()
}

// Release buffer to allow it to be recycled.
func (r *ReplayReader) Release(p []int16) {
	// Don't push down to the backing reader yet.
	// Either "ReadFrame" or "Close" will eventually release it.
}

// Allocate a new Payload.
func (r *ReplayReader) Alloc() []int16 {
	// Push down to backing reader.
	return r.reader.Alloc()
}

// ReadFrame the next Payload.
func (r *ReplayReader) ReadFrame() ([]int16, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, io.ErrClosedPipe
	}

	frame, err := r.reader.ReadFrame()
	if err != nil {
		return frame, err
	}

	idx := r.count % len(r.buffer)
	release := r.buffer[idx]
	r.buffer[idx] = frame

	// Release buffer?
	if release != nil {
		// Use the backing Reader for buffer management.
		r.reader.Release(release[:])
	}

	r.count++
	return frame, nil
}
