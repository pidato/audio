package vad

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/riff"
	"github.com/go-audio/wav"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"
)

type FrameSize int

func (f FrameSize) Duration() time.Duration {
	switch f {
	case 160:
		return time.Millisecond * 10
	case 320:
		return time.Millisecond * 20
	case 480:
		return time.Millisecond * 30
	}
	return time.Millisecond * 10
}

func TestVAD_Process(t *testing.T) {
	reader, err := OpenWav("recording.wav", 480)
	if err != nil {
		t.Fatal(err)
	}

	vad, _ := New(16000, VeryAggressive)

	frameDuration := reader.FrameDuration()
	speaking := false
	speakingAt := time.Duration(0)
	frameCount := 0

	_ = speakingAt

	fmt.Println("Silence")
	fmt.Printf("\tFrom: 	%v\n", time.Duration(0))

	for {
		frame, err := reader.Next()
		if err != nil {
			break
		}

		frameCount++
		elapsed := frameDuration * time.Duration(frameCount)
		result := vad.Process(frame)
		switch result {
		case Voice:
			if !speaking {
				fmt.Printf("\tEnd: 	%v\n", elapsed)
				fmt.Printf("\tDur: 	%v\n", elapsed-speakingAt)

				speaking = true
				fmt.Println("Speaking")
				fmt.Printf("\tFrom: 	%v\n", elapsed)
				speakingAt = elapsed
			}
		case Noise:
			if speaking {
				speaking = false
				fmt.Printf("\tEnd: 	%v\n", elapsed)
				fmt.Printf("\tDur: 	%v\n", elapsed-speakingAt)
				speakingAt = elapsed

				fmt.Println("Silence")
				fmt.Printf("\tFrom: 	%v\n", elapsed)
			}
		default:
			fmt.Println("Err", result)
		}
	}
}

func TestDownSampleBy2(t *testing.T) {
	frames20ms, err := load("recording.wav", 320)
	if err != nil {
		t.Fatal(err)
	}

	fout, err := os.OpenFile("8000.pcm", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)
	var state [8]int32
	frames8000 := make([][]int16, 0, 128)
	for _, frame := range frames20ms {
		out := make([]int16, 160)
		DownSampleBy2(frame, out, &state)
		frames8000 = append(frames8000, out)

		b := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&out[0])),
			Len:  len(out) * 2,
			Cap:  len(out) * 2,
		}))

		_, _ = fout.Write(b)
	}

	_ = fout.Sync()
	_ = fout.Close()
}

func TestUpSampleBy2(t *testing.T) {
	frames20ms, err := load("recording.wav", 320)
	if err != nil {
		t.Fatal(err)
	}

	fout, err := os.OpenFile("32000.pcm", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)
	var state [8]int32
	frames8000 := make([][]int16, 0, 128)
	for _, frame := range frames20ms {
		out := make([]int16, 640)
		UpSampleBy2(frame, out, &state)
		frames8000 = append(frames8000, out)

		b := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&out[0])),
			Len:  len(out) * 2,
			Cap:  len(out) * 2,
		}))

		_, _ = fout.Write(b)
	}

	_ = fout.Sync()
	_ = fout.Close()
}

func TestUpSample8000to16000(t *testing.T) {
	TestDownSampleBy2(t)
	all, err := ioutil.ReadFile("8000.pcm")
	if err != nil {
		t.Fatal(err)
	}

	fout, err := os.OpenFile("16000.pcm", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)

	var state [8]int32
	for i := 0; i < len(all); i += 320 {
		if i+320 > len(all) {
			break
		}
		inBytes := all[i : i+320]
		in := *(*[]int16)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&inBytes[0])),
			Len:  160,
			Cap:  160,
		}))

		out := make([]int16, 320)
		err = UpSampleBy2(in, out, &state)
		if err != nil {
			t.Fatal(err)
		}
		b := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&out[0])),
			Len:  len(out) * 2,
			Cap:  len(out) * 2,
		}))

		_, _ = fout.Write(b)
	}

	_ = fout.Sync()
	_ = fout.Close()
}

func TestVAD_SetMode(t *testing.T) {
	frames10ms, err := load("recording.wav", 160)
	if err != nil {
		t.Fatal(err)
	}
	frames20ms, err := load("recording.wav", 320)
	if err != nil {
		t.Fatal(err)
	}
	frames30ms, err := load("recording.wav", 480)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Quality-10ms", func(t *testing.T) {
		printStats(t, Quality, frames10ms)
	})
	t.Run("Quality-20ms", func(t *testing.T) {
		printStats(t, Quality, frames20ms)
	})
	t.Run("Quality-30ms", func(t *testing.T) {
		printStats(t, Quality, frames30ms)
	})
	t.Run("LowBitrate-10ms", func(t *testing.T) {
		printStats(t, LowBitrate, frames10ms)
	})
	t.Run("LowBitrate-20ms", func(t *testing.T) {
		printStats(t, LowBitrate, frames20ms)
	})
	t.Run("LowBitrate-30ms", func(t *testing.T) {
		printStats(t, LowBitrate, frames30ms)
	})
	t.Run("Aggressive-10ms", func(t *testing.T) {
		printStats(t, Aggressive, frames10ms)
	})
	t.Run("Aggressive-20ms", func(t *testing.T) {
		printStats(t, Aggressive, frames20ms)
	})
	t.Run("Aggressive-30ms", func(t *testing.T) {
		printStats(t, Aggressive, frames30ms)
	})
	t.Run("VeryAggressive-10ms", func(t *testing.T) {
		printStats(t, VeryAggressive, frames10ms)
	})
	t.Run("VeryAggressive-20ms", func(t *testing.T) {
		printStats(t, VeryAggressive, frames20ms)
	})
	t.Run("VeryAggressive-30ms", func(t *testing.T) {
		printStats(t, VeryAggressive, frames30ms)
	})
}

func BenchmarkVAD_Process(b *testing.B) {
	frames10ms, err := load("recording.wav", 160)
	if err != nil {
		b.Fatal(err)
	}
	frames20ms, err := load("recording.wav", 320)
	if err != nil {
		b.Fatal(err)
	}
	frames30ms, err := load("recording.wav", 480)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("Quality-10ms", func(b *testing.B) {
		runProcess(b, Quality, frames10ms)
	})
	b.Run("Quality-20ms", func(b *testing.B) {
		runProcess(b, Quality, frames20ms)
	})
	b.Run("Quality-30ms", func(b *testing.B) {
		runProcess(b, Quality, frames30ms)
	})
	b.Run("LowBitrate-10ms", func(b *testing.B) {
		runProcess(b, LowBitrate, frames10ms)
	})
	b.Run("LowBitrate-20ms", func(b *testing.B) {
		runProcess(b, LowBitrate, frames20ms)
	})
	b.Run("LowBitrate-30ms", func(b *testing.B) {
		runProcess(b, LowBitrate, frames30ms)
	})
	b.Run("Aggressive-10ms", func(b *testing.B) {
		runProcess(b, Aggressive, frames10ms)
	})
	b.Run("Aggressive-20ms", func(b *testing.B) {
		runProcess(b, Aggressive, frames20ms)
	})
	b.Run("Aggressive-30ms", func(b *testing.B) {
		runProcess(b, Aggressive, frames30ms)
	})
	b.Run("VeryAggressive-10ms", func(b *testing.B) {
		runProcess(b, VeryAggressive, frames10ms)
	})
	b.Run("VeryAggressive-20ms", func(b *testing.B) {
		runProcess(b, VeryAggressive, frames20ms)
	})
	b.Run("VeryAggressive-30ms", func(b *testing.B) {
		runProcess(b, VeryAggressive, frames30ms)
	})
}

func BenchmarkVAD_New(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vad, _ := New(16000, Aggressive)
		_ = vad.Close()
	}
}

func BenchmarkVAD_Reset(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vad, _ := New(16000, Aggressive)
		vad.Reset()
		_ = vad.Close()
	}
}

func runProcess(b *testing.B, mode Mode, data [][]int16) {
	for i := 0; i < b.N; i++ {
		vad, _ := New(16000, mode)
		defer vad.Close()
		//vad.SetMode(mode)
		//vad.SetSampleRate(16000)

		for frameNum, frame := range data {
			_ = frameNum
			switch vad.Process(frame) {
			case Voice:

			case Noise:

			default:
				fmt.Println("Err")
			}
		}
	}
}

type Stats struct {
	TotalSpeech  time.Duration
	TotalSilence time.Duration
}

func printStats(t *testing.T, mode Mode, data [][]int16) {
	stats := runProcessStats(t, mode, data)
	fmt.Printf("\t\tSpeech: 	%v\n", stats.TotalSpeech)
	fmt.Printf("\t\tSilence: 	%v\n", stats.TotalSilence)
	fmt.Println()
}

func runProcessStats(t *testing.T, mode Mode, data [][]int16) Stats {
	stats := Stats{}

	speaking := false
	elapsed := time.Duration(0)
	eventAt := time.Duration(0)
	frameDuration := FrameSize(len(data[0])).Duration()

	vad, _ := New(16000, mode)
	defer vad.Close()
	//vad.SetMode(mode)
	//vad.SetSampleRate(16000)

	for frameNum, frame := range data {
		elapsed = frameDuration * time.Duration(frameNum)

		switch vad.Process(frame) {
		case Voice:
			if !speaking {
				stats.TotalSilence += elapsed - eventAt
				speaking = true
				eventAt = elapsed
			}
		case Noise:
			if speaking {
				speaking = false
				stats.TotalSpeech += elapsed - eventAt
				eventAt = elapsed
			}

		default:
			fmt.Println("Err")
		}
	}

	return stats
}

func load(filename string, frameSize FrameSize) ([][]int16, error) {
	file, err := OpenWav("recording.wav", frameSize)
	if err != nil {
		return nil, err
	}

	var data [][]int16

	for {
		frame, err := file.Next()
		if err != nil {
			if len(frame) > 0 {
				data = append(data, frame)
			}
			break
		}
		data = append(data, frame)
	}

	return data, nil
}

type FrameReader struct {
	reader PCMReader

	frameSize int
	frames    [][]int16
	overflow  []int16
	eof       bool
}

func OpenWav(filename string, frameSize FrameSize) (*FrameReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	switch frameSize {
	case 80:
	case 160:
	case 320:
	default:
		frameSize = 160
	}

	reader := NewWavToPCM16(file)
	return &FrameReader{
		reader:    reader,
		frameSize: int(frameSize),
		frames:    nil,
		overflow:  nil,
		eof:       false,
	}, nil
}

func (f *FrameReader) FrameDuration() time.Duration {
	return FrameSize(f.frameSize).Duration()
}

func (f *FrameReader) Next() ([]int16, error) {
	for {
		if len(f.frames) > 0 {
			first := f.frames[0]
			f.frames = f.frames[1:]
			return first, nil
		}

		if f.eof {
			return nil, io.EOF
		}

		var err error
		f.overflow, err = f.reader.Read(f.overflow)

		if err == io.EOF {
			f.eof = true
		}

		if len(f.overflow) < f.frameSize {
			continue
		}

		split := len(f.overflow) / f.frameSize
		for i := 0; i < split; i++ {
			f.frames = append(f.frames, f.overflow[i*f.frameSize:i*f.frameSize+f.frameSize])
		}
		f.overflow = f.overflow[split*f.frameSize:]
	}
}

var (
	ErrPCMChunkNotFound = errors.New("PCM chunk not found")
	ErrPCMNot16Bit      = errors.New("PCM is not 16-bit")
	ErrClosed           = errors.New("closed")
)

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

	err := s.reader.Close()
	s.closed = true
	s.mu.Unlock()

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
	var buf []byte
	for int64(s.at) < offset {
		if buf == nil {
			buf = make([]byte, 8192)
		}
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

type WavToPCM struct {
	reader   io.ReadCloser
	decoder  *wav.Decoder
	didFirst bool
	closed   bool

	err error

	bytesPerSample uint16
	sampleBufData  []byte
	decodeFn       func(io.Reader, []byte) (int16, error)

	mu sync.Mutex
}

func NewWavToPCM16(reader io.ReadCloser) *WavToPCM {
	r := NewSeekReader(reader)
	return &WavToPCM{
		reader:         r,
		decoder:        wav.NewDecoder(r),
		didFirst:       false,
		bytesPerSample: 0,
		sampleBufData:  nil,
		decodeFn:       nil,
	}
}

func (w *WavToPCM) Close() error {
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

func (w *WavToPCM) IsClosed() bool {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	return closed
}

type PCMReader interface {
	Read(buffer []int16) ([]int16, error)
}

func (w *WavToPCM) Read(buffer []int16) ([]int16, error) {
	var chunk *riff.Chunk
	var err error
	if !w.didFirst {
		w.didFirst = true
		if !w.decoder.WasPCMAccessed() {
			err := w.decoder.FwdToPCM()
			if err != nil {
				return nil, w.decoder.Err()
			}
		}
		chunk = w.decoder.PCMChunk
		if chunk == nil {
			return nil, ErrPCMChunkNotFound
		}

		w.bytesPerSample = (w.decoder.BitDepth-1)/8 + 1
		w.sampleBufData = make([]byte, w.bytesPerSample)

		if w.decoder.BitDepth != 16 {

			return nil, ErrPCMNot16Bit
		}

		w.decodeFn, err = sampleDecodeFunc(int(w.decoder.BitDepth))
	} else {
		chunk, err = w.decoder.NextChunk()
		if err != nil {
			return nil, err
		}
	}

	var next int16
	for {
		next, err = w.decodeFn(chunk, w.sampleBufData)
		if err != nil {
			break
		}

		buffer = append(buffer, next)
	}

	return buffer, err
}

// sampleDecodeFunc returns a function that can be used to convert
// a byte range into an int value based on the amount of bits used per sample.
// Note that 8bit samples are unsigned, all other values are signed.
func sampleDecodeFunc(bitsPerSample int) (func(io.Reader, []byte) (int16, error), error) {
	// NOTE: WAV PCM data is stored using little-endian
	switch bitsPerSample {
	case 8:
		// 8bit values are unsigned
		return func(r io.Reader, buf []byte) (int16, error) {
			_, err := r.Read(buf[:1])
			return int16(buf[0]), err
		}, nil
	case 16:
		return func(r io.Reader, buf []byte) (int16, error) {
			_, err := r.Read(buf[:2])
			return int16(binary.LittleEndian.Uint16(buf[:2])), err
		}, nil
	case 24:
		// -34,359,738,367 (0x7FFFFF) to 34,359,738,368	(0x800000)
		return func(r io.Reader, buf []byte) (int16, error) {
			_, err := r.Read(buf[:3])
			if err != nil {
				return 0, err
			}
			return int16(audio.Int24LETo32(buf[:3])), nil
		}, nil
	case 32:
		return func(r io.Reader, buf []byte) (int16, error) {
			_, err := r.Read(buf[:4])
			return int16(int32(binary.LittleEndian.Uint32(buf[:4]))), err
		}, nil
	default:
		return nil, fmt.Errorf("unhandled byte depth:%d", bitsPerSample)
	}
}
