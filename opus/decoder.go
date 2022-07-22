// Copyright © 2015-2017 Go Opus Authors (see AUTHORS file)
//
// License for use of this code is detailed in the LICENSE file

package opus

import (
	"fmt"
	"unsafe"
)

/*
#include <opus.h>

int
bridge_decoder_get_last_packet_duration(OpusDecoder *st, opus_int32 *samples)
{
	return opus_decoder_ctl(st, OPUS_GET_LAST_PACKET_DURATION(samples));
}
*/
import "C"

var errDecUninitialized = fmt.Errorf("opus decoder uninitialized")

var (
	_empty [8192]byte
)

type Decoder struct {
	p *C.struct_OpusDecoder
	// Same purpose as encoder struct
	mem        []byte
	sampleRate int
	channels   int
}

// NewDecoder allocates a new Opus decoder and initializes it with the
// appropriate parameters. All related memory is managed by the Go GC.
func NewDecoder(sampleRate int, channels int) (*Decoder, error) {
	var dec Decoder
	err := dec.Init(sampleRate, channels)
	if err != nil {
		return nil, err
	}
	return &dec, nil
}

func (dec *Decoder) Init(sampleRate int, channels int) error {
	if dec.p != nil {
		return fmt.Errorf("opus decoder already initialized")
	}
	if channels != 1 && channels != 2 {
		return fmt.Errorf("Number of channels must be 1 or 2: %d", channels)
	}
	size := C.opus_decoder_get_size(C.int(channels))
	dec.sampleRate = sampleRate
	dec.channels = channels
	// Does the current buffer have enough room?
	if cap(dec.mem) < int(size) {
		dec.mem = make([]byte, size)
	} else {
		// memset to 0
		dec.mem = dec.mem[:size]
		if len(_empty) < len(dec.mem) {
			count := len(dec.mem) / len(_empty)
			for i := 0; i < count; i++ {
				copy(dec.mem[i*len(_empty):i*len(_empty)+len(_empty)], _empty[:])
			}
			last := len(dec.mem) % len(_empty)
			if last > 0 {
				copy(dec.mem[count*len(_empty):], _empty[:last])
			}
		} else {
			copy(dec.mem, _empty[:len(dec.mem)])
		}
	}

	dec.p = (*C.OpusDecoder)(unsafe.Pointer(&dec.mem[0]))

	errno := C.opus_decoder_init(
		dec.p,
		C.opus_int32(sampleRate),
		C.int(channels))
	if errno != 0 {
		return Error(errno)
	}
	return nil
}

// Decode encoded Opus data into the supplied buffer. On success, returns the
// number of samples correctly written to the target buffer.
func (dec *Decoder) Decode(data []byte, pcm []int16) (int, error) {
	if dec.p == nil {
		return 0, errDecUninitialized
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("opus: no data supplied")
	}
	if len(pcm) == 0 {
		return 0, fmt.Errorf("opus: target buffer empty")
	}
	if cap(pcm)%dec.channels != 0 {
		return 0, fmt.Errorf("opus: target buffer capacity must be multiple of channels")
	}
	n := int(C.opus_decode(
		dec.p,
		(*C.uchar)(&data[0]),
		C.opus_int32(len(data)),
		(*C.opus_int16)(&pcm[0]),
		C.int(cap(pcm)/dec.channels),
		0))
	if n < 0 {
		return 0, Error(n)
	}
	return n, nil
}

// Decode encoded Opus data into the supplied buffer. On success, returns the
// number of samples correctly written to the target buffer.
func (dec *Decoder) DecodeFloat32(data []byte, pcm []float32) (int, error) {
	if dec.p == nil {
		return 0, errDecUninitialized
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("opus: no data supplied")
	}
	if len(pcm) == 0 {
		return 0, fmt.Errorf("opus: target buffer empty")
	}
	if cap(pcm)%dec.channels != 0 {
		return 0, fmt.Errorf("opus: target buffer capacity must be multiple of channels")
	}
	n := int(C.opus_decode_float(
		dec.p,
		(*C.uchar)(&data[0]),
		C.opus_int32(len(data)),
		(*C.float)(&pcm[0]),
		C.int(cap(pcm)/dec.channels),
		0))
	if n < 0 {
		return 0, Error(n)
	}
	return n, nil
}

// DecodeFEC encoded Opus data into the supplied buffer with forward error
// correction. It is to be used on the packet directly following the lost one.
// The supplied buffer needs to be exactly the duration of audio that is missing
func (dec *Decoder) DecodeFEC(data []byte, pcm []int16) error {
	if dec.p == nil {
		return errDecUninitialized
	}
	if len(data) == 0 {
		return fmt.Errorf("opus: no data supplied")
	}
	if len(pcm) == 0 {
		return fmt.Errorf("opus: target buffer empty")
	}
	if cap(pcm)%dec.channels != 0 {
		return fmt.Errorf("opus: target buffer capacity must be multiple of channels")
	}
	n := int(C.opus_decode(
		dec.p,
		(*C.uchar)(&data[0]),
		C.opus_int32(len(data)),
		(*C.opus_int16)(&pcm[0]),
		C.int(cap(pcm)/dec.channels),
		1))
	if n < 0 {
		return Error(n)
	}
	return nil
}

// DecodeFECFloat32 encoded Opus data into the supplied buffer with forward error
// correction. It is to be used on the packet directly following the lost one.
// The supplied buffer needs to be exactly the duration of audio that is missing
func (dec *Decoder) DecodeFECFloat32(data []byte, pcm []float32) error {
	if dec.p == nil {
		return errDecUninitialized
	}
	if len(data) == 0 {
		return fmt.Errorf("opus: no data supplied")
	}
	if len(pcm) == 0 {
		return fmt.Errorf("opus: target buffer empty")
	}
	if cap(pcm)%dec.channels != 0 {
		return fmt.Errorf("opus: target buffer capacity must be multiple of channels")
	}
	n := int(C.opus_decode_float(
		dec.p,
		(*C.uchar)(&data[0]),
		C.opus_int32(len(data)),
		(*C.float)(&pcm[0]),
		C.int(cap(pcm)/dec.channels),
		1))
	if n < 0 {
		return Error(n)
	}
	return nil
}

// LastPacketDuration gets the duration (in samples)
// of the last packet successfully decoded or concealed.
func (dec *Decoder) LastPacketDuration() (int, error) {
	var samples C.opus_int32
	res := C.bridge_decoder_get_last_packet_duration(dec.p, &samples)
	if res != C.OPUS_OK {
		return 0, Error(res)
	}
	return int(samples), nil
}
