package stream

import (
	"github.com/pidato/audio/opus"
	"github.com/pidato/audio/pcm"
	"os"
	"sync"
)

type Decoder struct {
	dec *opus.Decoder
	pcm *pcm.Buffer

	sampleRate      int
	channels        int
	ptime           int
	samplesPerFrame int
	pcmBuf          []int16
	next            []int16
	nextLen         int

	closed bool
	mu     sync.Mutex
}

func NewDecoder(sampleRate, ptime, maxFrames int) (*Decoder, error) {
	d, err := opus.NewDecoder(sampleRate, 1)
	if err != nil {
		return nil, err
	}
	buf, err := pcm.NewBuffer(sampleRate, ptime, maxFrames)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		dec:             d,
		pcm:             buf,
		sampleRate:      sampleRate,
		channels:        1,
		ptime:           ptime,
		samplesPerFrame: buf.FrameSize(),
		pcmBuf:          make([]int16, 2880*2),
		next:            nil,
		nextLen:         0,
	}, nil
}

func (d *Decoder) Decoder() *opus.Decoder {
	return d.dec
}

func (d *Decoder) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	err := d.pcm.Close()
	if err != nil {
		return err
	}
	return nil
}

func (d *Decoder) Reset() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.dec.Init(d.sampleRate, d.channels)
}

func (d *Decoder) Drop(frames int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := 0; i < frames; i++ {

	}
	return nil
}

func (d *Decoder) Write(packet []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var samples int
	var err error

	samples, err = d.dec.Decode(packet, d.pcmBuf)
	if err != nil {
		return err
	}
	if samples < 0 {
		return os.ErrInvalid
	}

	samplesPerFrame := d.samplesPerFrame

	buf := d.pcmBuf[0:samples]
	next := d.next

	if d.nextLen > 0 {
		next := d.next
		remaining := samplesPerFrame - d.nextLen
		if remaining > samples {
			copy(next[d.nextLen:], buf[0:samples])
			d.nextLen += samples
			// Not enough for the next frame.
			return nil
		} else if remaining == samples {
			copy(next[d.nextLen:], buf[0:samples])
			d.nextLen = 0
			return d.pcm.Write(next)
		} else {
			copy(next[d.nextLen:], buf[0:remaining])
			d.nextLen = 0

			err = d.pcm.Write(next)
			if err != nil {
				return err
			}

			d.next = d.pcm.Alloc()
			buf = buf[remaining:]
		}
	} else if d.next == nil {
		next = d.pcm.Alloc()
	}

	for len(buf) > 0 {
		if len(buf) > samplesPerFrame {
			copy(next, buf[0:samplesPerFrame])
			err = d.pcm.Write(next)
			if err != nil {
				return err
			}
			buf = buf[samplesPerFrame:]
		} else if len(buf) == samplesPerFrame {
			copy(next, buf)
			return d.pcm.Write(next)
		} else {
			copy(next, buf)
			d.next = next
			d.nextLen = len(buf)
			return nil
		}
	}

	return nil
}

func (d *Decoder) WriteEOF() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.nextLen > 0 {
		next := d.next
		for i := d.nextLen; i < len(next); i++ {
			next[i] = 0
		}
		d.nextLen = 0
		d.next = nil
		if err := d.pcm.Write(next); err != nil {
			return err
		}
	}
	return d.pcm.WriteFinal()
}

func (d *Decoder) ReadFrame() ([]int16, error) {
	return d.pcm.ReadFrame()
}
