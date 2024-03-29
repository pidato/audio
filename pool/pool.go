package pool

import (
	"errors"
	"github.com/bytedance/gopkg/lang/syncx"
	"github.com/pidato/audio/util"
)

var (
	ErrUnsupported = errors.New("unsupported")
)

const (
	frameSize48khz2dot5ms = 48000 / (1000 / 2.5)
	frameSize48khz5ms     = frameSize48khz2dot5ms * 2
	frameSize48khz10ms    = frameSize48khz5ms * 2
	frameSize48khz20ms    = frameSize48khz10ms * 2
	frameSize48khz40ms    = frameSize48khz20ms * 2
	frameSize48khz60ms    = frameSize48khz20ms * 3
	frameSize48khz120ms   = frameSize48khz60ms * 2
)

var (
	Pool8kHz  = newPool(8000)
	Pool12kHz = newPool(12000)
	Pool16kHz = newPool(16000)
	Pool24kHz = newPool(24000)
	Pool48kHz = newPool(48000)

	Opus2dot5ms = newPCMPool(48000, frameSize48khz2dot5ms, nil)
	Opus5ms     = newPCMPool(48000, frameSize48khz5ms, nil)
	Opus10ms    = newPCMPool(48000, frameSize48khz10ms, nil)
	Opus20ms    = newPCMPool(48000, frameSize48khz20ms, nil)
	Opus40ms    = newPCMPool(48000, frameSize48khz40ms, nil)
	Opus60ms    = newPCMPool(48000, frameSize48khz60ms, nil)
	Opus120ms   = newPCMPool(48000, frameSize48khz120ms, nil)

	Bytes16    = newBytes(16)
	Bytes24    = newBytes(24)
	Bytes32    = newBytes(32)
	Bytes48    = newBytes(48)
	Bytes64    = newBytes(64)
	Bytes96    = newBytes(96)
	Bytes128   = newBytes(128)
	Bytes256   = newBytes(256)
	Bytes512   = newBytes(512)
	Bytes768   = newBytes(768)
	Bytes1024  = newBytes(1024)
	Bytes1500  = newBytes(1500)
	Bytes2048  = newBytes(2048)
	Bytes4096  = newBytes(4096)
	Bytes8192  = newBytes(8192)
	Bytes16384 = newBytes(16384)
	Bytes32768 = newBytes(32768)
	Bytes65535 = newBytes(65535)

	Bytes160 = newBytes(160)
	Bytes240 = newBytes(240)
	Bytes320 = newBytes(320)
	Bytes480 = newBytes(480)
	Bytes640 = newBytes(640)
	Bytes960 = newBytes(960)
)

type Pool struct {
	ClockSpeed int
	Multiple   int
	PCM2dot5   *PCM
	PCM5ms     *PCM
	PCM10ms    *PCM
	PCM20ms    *PCM
	PCM40ms    *PCM
	PCM60ms    *PCM
	PCM120ms   *PCM
}

func newPool(clockSpeed int) *Pool {
	multiple := 48000 / clockSpeed
	if 48000%clockSpeed != 0 {
		panic("clockSpeed not multiple of 48000")
	}

	return &Pool{
		ClockSpeed: clockSpeed,
		Multiple:   multiple,
		PCM2dot5:   newPCMPool(clockSpeed, frameSize48khz2dot5ms/multiple, Opus2dot5ms),
		PCM5ms:     newPCMPool(clockSpeed, frameSize48khz5ms/multiple, Opus5ms),
		PCM10ms:    newPCMPool(clockSpeed, frameSize48khz10ms/multiple, Opus10ms),
		PCM20ms:    newPCMPool(clockSpeed, frameSize48khz20ms/multiple, Opus20ms),
		PCM40ms:    newPCMPool(clockSpeed, frameSize48khz40ms/multiple, Opus40ms),
		PCM60ms:    newPCMPool(clockSpeed, frameSize48khz60ms/multiple, Opus60ms),
		PCM120ms:   newPCMPool(clockSpeed, frameSize48khz120ms/multiple, Opus120ms),
	}
}

func (p *Pool) ForPtime(ptime int) *PCM {
	switch ptime {
	case 2:
		return p.PCM2dot5
	case 3:
		return p.PCM2dot5
	case 5:
		return p.PCM5ms
	case 10:
		return p.PCM10ms
	case 20:
		return p.PCM20ms
	case 40:
		return p.PCM40ms
	case 60:
		return p.PCM60ms
	case 120:
		return p.PCM120ms
	}
	return p.PCM20ms
}

func Of(sampleRate, ptime int) (*Pool, error) {
	switch ptime {
	case 3:
	case 5:
	case 10:
	case 20:
	case 40:
	case 60:
	default:
		return nil, ErrUnsupported
	}
	switch sampleRate {
	case 8000:
		return Pool8kHz, nil
	case 12000:
		return Pool12kHz, nil
	case 16000:
		return Pool16kHz, nil
	case 24000:
		return Pool24kHz, nil
	case 48000:
		return Pool48kHz, nil
	}
	return nil, ErrUnsupported
}

func OpusFrameSizeOf(ptime int) int {
	switch ptime {
	case 3:
		return frameSize48khz2dot5ms
	case 5:
		return frameSize48khz5ms
	case 10:
		return frameSize48khz10ms
	case 20:
		return frameSize48khz20ms
	case 40:
		return frameSize48khz40ms
	case 60:
		return frameSize48khz60ms
	case 120:
		return frameSize48khz120ms
	}
	return 0
}

type PCM struct {
	ClockSpeed int
	FrameSize  int
	Opus       *PCM
	pool       syncx.Pool
}

func newPCMPool(clockSpeed, size int, opus *PCM) *PCM {
	p := &PCM{
		ClockSpeed: clockSpeed,
		FrameSize:  size,
		Opus:       opus,
		pool: syncx.Pool{New: func() interface{} {
			return make([]int16, size)
		}},
	}
	if opus == nil {
		p.Opus = opus
	}
	return p
}

func (p *PCM) Get() []int16 {
	return p.pool.Get().([]int16)
}

func (p *PCM) Release(pcm []int16) {
	if cap(pcm) < p.FrameSize {
		return
	}
	pcm = pcm[:p.FrameSize]
	p.pool.Put(pcm)
}

type Bytes struct {
	size int
	pool syncx.Pool
}

func (b *Bytes) Size() int {
	return b.size
}

func newBytes(size int) *Bytes {
	p := &Bytes{
		size: size,
		pool: syncx.Pool{New: func() interface{} {
			return make([]byte, size, size)
		}, NoGC: true},
	}
	return p
}

func (p *Bytes) Get() []byte {
	return p.pool.Get().([]byte)
}

func (p *Bytes) Release(buf []byte) {
	if cap(buf) < p.size {
		return
	}
	buf = buf[0:p.size]
	p.pool.Put(buf)
}

func U8Get(size int) []byte {
	switch size {
	case 8:
		return Bytes16.Get()[0:8]
	case 10:
		return Bytes16.Get()[0:10]
	case 12:
		return Bytes16.Get()[0:12]
	case 16:
		return Bytes16.Get()
	case 20:
		return Bytes24.Get()[0:20]
	case 24:
		return Bytes24.Get()
	case 30:
		return Bytes32.Get()[0:30]
	case 32:
		return Bytes32.Get()
	case 40:
		return Bytes48.Get()[0:40]
	case 48:
		return Bytes48.Get()
	case 50:
		return Bytes64.Get()[0:50]
	case 60:
		return Bytes64.Get()[0:60]
	case 64:
		return Bytes64.Get()
	case 70:
		return Bytes96.Get()[0:70]
	case 72:
		return Bytes96.Get()[0:72]
	case 80:
		return Bytes96.Get()[0:80]
	case 90:
		return Bytes96.Get()[0:90]
	case 96:
		return Bytes96.Get()
	case 100:
		return Bytes128.Get()[0:100]
	case 128:
		return Bytes128.Get()
	case 160:
		return Bytes160.Get()
	case 256:
		return Bytes256.Get()
	case 320:
		return Bytes320.Get()
	case 480:
		return Bytes480.Get()
	case 512:
		return Bytes512.Get()
	case 640:
		return Bytes640.Get()
	case 768:
		return Bytes768.Get()
	case 960:
		return Bytes960.Get()
	case 1024:
		return Bytes1024.Get()
	case 1500:
		return Bytes1500.Get()
	case 2048:
		return Bytes2048.Get()
	case 4096:
		return Bytes4096.Get()
	case 8192:
		return Bytes8192.Get()
	case 16384:
		return Bytes16384.Get()
	case 32768:
		return Bytes32768.Get()
	case 65535:
		return Bytes65535.Get()
	}
	if size < 8 {
		return Bytes16.Get()[0:size]
	}
	return make([]byte, size, size)
}

func U8Release(b []byte) {
	switch cap(b) {
	case 16:
		Bytes16.Release(b)
	case 24:
		Bytes24.Release(b)
	case 32:
		Bytes32.Release(b)
	case 48:
		Bytes48.Release(b)
	case 64:
		Bytes64.Release(b)
	case 96:
		Bytes96.Release(b)
	case 128:
		Bytes128.Release(b)
	case 160:
		Bytes160.Release(b)
	case 256:
		Bytes256.Release(b)
	case 320:
		Bytes320.Release(b)
	case 480:
		Bytes480.Release(b)
	case 512:
		Bytes512.Release(b)
	case 640:
		Bytes640.Release(b)
	case 768:
		Bytes768.Release(b)
	case 960:
		Bytes960.Release(b)
	case 1024:
		Bytes1024.Release(b)
	case 1500:
		Bytes1500.Release(b)
	case 2048:
		Bytes2048.Release(b)
	case 4096:
		Bytes4096.Release(b)
	case 8192:
		Bytes8192.Release(b)
	case 16384:
		Bytes16384.Release(b)
	case 32768:
		Bytes32768.Release(b)
	case 65535:
		Bytes65535.Release(b)
	}
}

func I16Get(size int) []int16 {
	switch size {
	case 80:
		return util.CastBytesToInt16(Bytes160.Get())
	case 160:
		return util.CastBytesToInt16(Bytes320.Get())
	case 320:
		return util.CastBytesToInt16(Bytes640.Get())
	case 480:
		return util.CastBytesToInt16(Bytes960.Get())
	case 640:
		return util.CastBytesToInt16(Bytes1500.Get())[0 : 640*2]
	case 960:
		return util.CastBytesToInt16(Bytes2048.Get())[0 : 960*2]
	}
	return util.CastBytesToInt16(U8Get(size * 2))[0:size]
}

func I16Release(b []int16) {
	U8Release(util.CastInt16ToBytes(b))
}
