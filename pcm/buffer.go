package pcm

import (
	"errors"
	"github.com/pidato/audio/pool"
	"io"
	"sync"
	"time"
)

// Buffer between Reader and Writer. Once internal frame buffer is full, it blocks
// until either the Reader or Writer processes the next frame.
type Buffer struct {
	sampleRate     int
	ptime          int
	pool           *pool.Pool
	pcmPool        *pool.PCM
	eof            bool
	max            int
	frames         [][]int16
	readerIndex    int
	writerIndex    int
	size           int
	maxDivergence  int
	writesBlocked  int
	samplesRead    int
	sampleDuration time.Duration
	closed         bool
	readerWait     bool
	writerWait     bool
	readerWg       sync.WaitGroup
	writerWg       sync.WaitGroup
	mu             sync.Mutex
}

func NewBuffer(sampleRate, ptime, maxFrames int) (*Buffer, error) {
	if maxFrames < 1 {
		maxFrames = 1
	}
	if maxFrames > 10000 {
		maxFrames = 10000
	}
	p, err := pool.Of(sampleRate, ptime)
	if err != nil {
		return nil, err
	}

	pcmPool := p.ForPtime(ptime)

	b := &Buffer{
		sampleRate:     sampleRate,
		ptime:          ptime,
		pool:           p,
		pcmPool:        pcmPool,
		eof:            false,
		max:            maxFrames,
		frames:         make([][]int16, maxFrames),
		samplesRead:    0,
		sampleDuration: time.Second / time.Duration(sampleRate),
		readerIndex:    0,
		writerIndex:    0,
		size:           0,
	}
	return b, nil
}

func (b *Buffer) MaxDivergenceFrames() int {
	return b.maxDivergence
}

func (b *Buffer) MaxDivergence() time.Duration {
	return time.Duration(b.maxDivergence) * time.Millisecond * time.Duration(b.ptime)
}

func (b *Buffer) Elapsed() time.Duration {
	return time.Duration(b.samplesRead) * b.sampleDuration
}

func (f *Buffer) SampleRate() int {
	return f.sampleRate
}

func (f *Buffer) FrameSize() int {
	return f.pcmPool.FrameSize
}

func (f *Buffer) Ptime() time.Duration {
	return time.Duration(f.ptime) * time.Millisecond
}

func (f *Buffer) Alloc() []int16 {
	return f.pcmPool.Get()
}

func (f *Buffer) Release(b []int16) {
	f.pcmPool.Release(b)
}

func (r *Buffer) Reset() error {
	r.mu.Lock()
	if r.closed {
		r.closed = false
		r.reset()
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	err := r.Close()
	if err != nil {
		return err
	}

	r.mu.Lock()
	if !r.closed {
		r.mu.Unlock()
		return errors.New("race")
	}
	r.reset()
	r.mu.Unlock()
	return nil
}

func (r *Buffer) reset() {
	r.eof = false
	r.size = 0
	r.readerIndex = 0
	r.writerIndex = 0
	r.frames = r.frames[0:0]
}

func (f *Buffer) Close() error {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return io.ErrClosedPipe
	}
	f.closed = true
	if f.readerWait {
		f.readerWait = false
		// Unblock readers.
		f.readerWg.Done()
	}
	if f.writerWait {
		// Unblock writer.
		f.writerWait = false
		f.writerWg.Done()
	}
	// Release frames.
	for i, buf := range f.frames {
		f.pcmPool.Release(buf)
		f.frames[i] = nil
	}
	f.frames = nil
	f.mu.Unlock()
	return nil
}

func (f *Buffer) WriteFinal() error {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return io.ErrClosedPipe
	}
	f.eof = true
	writerWait := f.writerWait
	if f.writerWait {
		f.writerWait = false
	}
	readerWait := f.readerWait
	if f.readerWait {
		f.readerWait = false
	}
	f.mu.Unlock()

	if writerWait {
		f.writerWg.Done()
	}
	if readerWait {
		f.readerWg.Done()
	}
	return nil
}

func (f *Buffer) Write(p []int16) error {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return io.ErrClosedPipe
	}
	if f.eof {
		f.mu.Unlock()
		return io.EOF
	}
	if f.size == len(f.frames) {
		f.mu.Unlock()
		return io.ErrShortBuffer
	}

	f.frames[f.writerIndex%len(f.frames)] = p
	f.writerIndex++
	f.size++
	if f.maxDivergence < f.size {
		f.maxDivergence = f.size
	}

	// Unblock reader.
	readerWait := f.readerWait
	if f.readerWait {
		f.readerWait = false
	}
	f.mu.Unlock()

	if readerWait {
		f.readerWg.Done()
	}
	return nil
}

func (f *Buffer) UnblockWriter() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writerWait {
		f.writerWait = false
		f.writerWg.Done()
		return true
	}
	return false
}

func (f *Buffer) WriteBlocking(p []int16) error {
	f.mu.Lock()
	// Was it recently closed.
	if f.closed {
		f.mu.Unlock()
		return io.ErrClosedPipe
	}
	if f.eof {
		f.mu.Unlock()
		return io.EOF
	}
	if f.size == f.max {
		// Wait for reader to read next frame.
		if !f.writerWait {
			f.writerWait = true
			f.writerWg.Add(1)
		}
		f.writesBlocked++
		f.mu.Unlock()
		// Wait for next ReadFrame to free up a slot.
		f.writerWg.Wait()
		// Try Non-Blocking write. This may return error.
		return f.Write(p)
	}
	f.frames[f.writerIndex%f.max] = p
	f.writerIndex++
	f.size++
	if f.maxDivergence < f.size {
		f.maxDivergence = f.size
	}

	readerWait := f.readerWait
	if readerWait {
		f.readerWait = false
	}

	f.mu.Unlock()

	if readerWait {
		f.readerWg.Done()
	}
	return nil
}

func (f *Buffer) ReadFrame() ([]int16, error) {
	for {
		f.mu.Lock()
		if f.closed {
			f.mu.Unlock()
			return nil, io.ErrClosedPipe
		}

		if f.size == 0 {
			if f.eof {
				f.mu.Unlock()
				return nil, io.EOF
			}

			// Wait for next write.
			if !f.readerWait {
				f.readerWait = true
				f.readerWg.Add(1)
			}
			f.mu.Unlock()
			f.readerWg.Wait()
			continue
		}

		buf := f.frames[f.readerIndex%f.max]
		f.readerIndex++
		f.size--
		f.samplesRead += len(buf)

		// Notify writer if needed.
		if f.writerWait {
			f.writerWait = false
			f.writerWg.Done()
		}
		f.mu.Unlock()

		return buf, nil
	}
}
