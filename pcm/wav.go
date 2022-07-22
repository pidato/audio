package pcm

import (
	"errors"
	"github.com/go-audio/riff"
	"github.com/go-audio/wav"
	"io"
	"os"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/pidato/audio/pool"
)

var (
	ErrNoChunk  = errors.New("no chunk")
	ErrNot16bit = errors.New("not 16bit")
)

func OpenWavFile(filename string, ptime int) (*WavReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader, err := OpenWav(file, ptime)
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	return reader, nil
}

type WavReader struct {
	reader         io.ReadCloser
	decoder        *wav.Decoder
	didFirst       bool
	closed         bool
	ptime          int
	err            error
	pool           *pool.Pool
	pcmPool        *pool.PCM
	sampleRate     int
	bytesPerSample uint16
	samplesRead    int
	sampleDuration time.Duration
	chunk          *riff.Chunk
	mu             sync.Mutex
}

func OpenWav(reader io.ReadCloser, ptime int) (*WavReader, error) {
	r := NewSeekReader(reader)
	w := &WavReader{
		reader:         r,
		decoder:        wav.NewDecoder(r),
		ptime:          ptime,
		didFirst:       false,
		bytesPerSample: 0,
	}

	if !w.decoder.WasPCMAccessed() {
		err := w.decoder.FwdToPCM()
		if err != nil {
			_ = w.Close()
			return nil, w.decoder.Err()
		}
	}
	w.chunk = w.decoder.PCMChunk
	if w.chunk == nil {
		_ = w.Close()
		return nil, ErrNoChunk
	}

	w.sampleRate = int(w.decoder.SampleRate)
	w.bytesPerSample = (w.decoder.BitDepth-1)/8 + 1
	w.sampleDuration = time.Second / time.Duration(w.sampleRate)

	if w.decoder.BitDepth != 16 {
		_ = w.Close()
		return nil, ErrNot16bit
	}

	var err error
	w.pool, err = pool.Of(w.sampleRate, ptime)
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	w.pcmPool = w.pool.ForPtime(ptime)

	return w, nil
}

func (w *WavReader) WavAudioFormat() uint16 {
	return w.decoder.WavAudioFormat
}

func (w *WavReader) Elapsed() time.Duration {
	return time.Duration(w.samplesRead) * w.sampleDuration
}

func (w *WavReader) Close() error {
	w.mu.Lock()
	if w.closed {
		err := w.err
		w.mu.Unlock()
		return err
	}
	w.closed = true
	reader := w.reader
	w.reader = nil
	err := w.err
	w.mu.Unlock()

	_ = reader.Close()
	return err
}

func (w *WavReader) IsClosed() bool {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	return closed
}

func (f *WavReader) SampleRate() int {
	return f.sampleRate
}

func (f *WavReader) FrameSize() int {
	return f.pcmPool.FrameSize
}

func (w *WavReader) Ptime() time.Duration {
	return time.Duration(w.ptime) * time.Millisecond
}

func (w *WavReader) Alloc() []int16 {
	return w.pcmPool.Get()
}

func (w *WavReader) Release(p []int16) {
	w.pcmPool.Release(p)
}

func (w *WavReader) ReadFrame() ([]int16, error) {
	buf := w.pcmPool.Get()
	n, err := w.Read(buf)
	if n <= 0 {
		w.pcmPool.Release(buf)
		return nil, err
	}
	if len(buf) != n {
		return buf[:n], err
	}
	return buf, err
}

func (w *WavReader) Read(buffer []int16) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(buffer) == 0 {
		return 0, io.ErrShortBuffer
	}

	frameBuffer := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: (uintptr)(unsafe.Pointer(&buffer[0])),
		Len:  len(buffer) * 2,
		Cap:  len(buffer) * 2,
	}))
	buf := frameBuffer
	count := 0

	for {
		if w.chunk == nil || w.chunk.IsFullyRead() {
			w.chunk, err = w.decoder.NextChunk()
			if err != nil {
				if err.Error() == "error reading chunk header - EOF" {
					return n, io.EOF
				}
				return n, err
			}
		}

		var read int
		read, err = w.chunk.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return n, err
		}
		if read == 0 {
			err = io.EOF
			continue
		}

		buf = frameBuffer[count:read]
		count += read
		n += read / 2
		w.samplesRead += n

		if len(buf) == cap(buf) {
			return n, nil
		}
	}
}

// sampleDecodeFunc returns a function that can be used to convert
// a byte range into an int value based on the amount of bits used per sample.
// Note that 8bit samples are unsigned, all other values are signed.
//func sampleDecodeFunc(bitsPerSample int) (func(io.Reader, []byte) (int16, error), error) {
//	// NOTE: WAV PCM data is stored using little-endian
//	switch bitsPerSample {
//	case 8:
//		// 8bit values are unsigned
//		return func(r io.Reader, pool []byte) (int16, error) {
//			_, err := r.ReadFrame(pool[:1])
//			return int16(pool[0]), err
//		}, nil
//	case 16:
//		return func(r io.Reader, pool []byte) (int16, error) {
//			_, err := r.ReadFrame(pool[:2])
//			return int16(binary.LittleEndian.Uint16(pool[:2])), err
//		}, nil
//	case 24:
//		// -34,359,738,367 (0x7FFFFF) to 34,359,738,368	(0x800000)
//		return func(r io.Reader, pool []byte) (int16, error) {
//			_, err := r.ReadFrame(pool[:3])
//			if err != nil {
//				return 0, err
//			}
//			return int16(audio.Int24LETo32(pool[:3])), nil
//		}, nil
//	case 32:
//		return func(r io.Reader, pool []byte) (int16, error) {
//			_, err := r.ReadFrame(pool[:4])
//			return int16(int32(binary.LittleEndian.Uint32(pool[:4]))), err
//		}, nil
//	default:
//		return nil, fmt.Errorf("unhandled byte depth:%d", bitsPerSample)
//	}
//}

var (
	_discardPool = &discardPool{pool: sync.Pool{New: func() interface{} {
		return make([]byte, 8192)
	}}}
)

type discardPool struct {
	pool sync.Pool
}

func (d *discardPool) Get() []byte {
	return d.pool.Get().([]byte)
}

func (d *discardPool) Put(p []byte) {
	d.pool.Put(p)
}

type SeekReader struct {
	io.ReadSeeker
	io.Closer
	reader io.ReadCloser
	at     uint64
	closed bool
	mu     sync.Mutex
}

func NewSeekReader(reader io.ReadCloser) *SeekReader {
	return &SeekReader{
		reader: reader,
		at:     0,
	}
}

func (s *SeekReader) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true
	s.mu.Unlock()

	err := s.reader.Close()
	return err
}

func (s *SeekReader) Read(b []byte) (int, error) {
	n, err := s.reader.Read(b)
	if err != nil {
		return n, err
	}

	s.at += uint64(n)
	return n, err
}

func (s *SeekReader) Seek(offset int64, whence int) (int64, error) {
	i := int64(0)
	discard := _discardPool.Get()
	defer _discardPool.Put(discard)

	buf := discard
	for int64(s.at) < offset {
		next := offset - int64(s.at)
		if next < int64(len(buf)) {
			buf = buf[:next]
		}

		n, err := s.reader.Read(buf)
		s.at += uint64(n)
		i += int64(n)
		if err != nil {
			return i, err
		}
	}
	return i, nil
}
