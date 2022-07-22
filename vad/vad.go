package vad

/*
#include <stdlib.h>
#include <string.h>
#include <inttypes.h>
#include "webrtc_vad.h"
#include "signal_processing_library.h"

typedef struct do_webrtc_vad_create_t {
	size_t inst;
	int32_t mode;
	int32_t result;
} do_webrtc_vad_create_t;

void do_webrtc_vad_create(size_t arg0, size_t arg1) {
	do_webrtc_vad_create_t* args = (do_webrtc_vad_create_t*)(void*)arg0;
	VadInst* inst = WebRtcVad_Create();
	args->inst = (size_t)(void*)inst;
	args->result = (int32_t)WebRtcVad_Init(inst);
	WebRtcVad_set_mode(inst, (int)args->mode);
}

typedef struct do_webrtc_vad_reset_t {
	size_t inst;
	int32_t mode;
	int32_t result;
} do_webrtc_vad_reset_t;

void do_webrtc_vad_reset(size_t arg0, size_t arg1) {
	do_webrtc_vad_reset_t* args = (do_webrtc_vad_reset_t*)(void*)arg0;
	WebRtcVad_Free((VadInst*)(void*)args->inst);
	VadInst* inst = WebRtcVad_Create();
	args->inst = (size_t)(void*)inst;
	args->result = (int32_t)WebRtcVad_Init(inst);
	WebRtcVad_set_mode(inst, (int)args->mode);
}

typedef struct do_webrtc_vad_free_t {
	size_t ptr;
} do_webrtc_vad_free_t;

void do_webrtc_vad_free(size_t arg0, size_t arg1) {
	do_webrtc_vad_free_t* args = (do_webrtc_vad_free_t*)(void*)arg0;
	//fvad_free((Fvad*)(void*)args->ptr);
	WebRtcVad_Free((VadInst*)(void*)args->ptr);
}

typedef struct do_webrtc_vad_process_t {
	size_t ptr;
	size_t data;
	size_t length;
	int32_t sampleRate;
	int32_t result;
} do_webrtc_vad_process_t;

void do_webrtc_vad_process(size_t arg0, size_t arg1) {
	do_webrtc_vad_process_t* args = (do_webrtc_vad_process_t*)(void*)arg0;
	args->result = (int32_t)WebRtcVad_Process(
		(VadInst*)(void*)args->ptr,
		(int)args->sampleRate,
		(int16_t*)(void*)args->data,
		args->length
	);
}

typedef struct do_webrtc_downsample_by_2_t {
	size_t in;
	size_t size;
	size_t out;
	size_t state;
} do_webrtc_downsample_by_2_t;

void do_webrtc_downsample_by_2(size_t arg0, size_t arg1) {
	do_webrtc_downsample_by_2_t* args = (do_webrtc_downsample_by_2_t*)(void*)arg0;
	WebRtcSpl_DownsampleBy2(
		(int16_t*)(void*)args->in,
		args->size,
		(int16_t*)(void*)args->out,
		(int32_t*)(void*)args->state
	);
}

typedef struct do_webrtc_upsample_by_2_t {
	size_t in;
	size_t size;
	size_t out;
	size_t state;
} do_webrtc_upsample_by_2_t;

void do_webrtc_upsample_by_2(size_t arg0, size_t arg1) {
	do_webrtc_upsample_by_2_t* args = (do_webrtc_upsample_by_2_t*)(void*)arg0;
	WebRtcSpl_UpsampleBy2(
		(int16_t*)(void*)args->in,
		args->size,
		(int16_t*)(void*)args->out,
		(int32_t*)(void*)args->state
	);
}

*/
import "C"
import (
	"errors"
	"github.com/pidato/unsafe/cgo"
	"reflect"
	"sync"
	"unsafe"
)

type State int

const (
	Voice   = State(1)
	Noise   = State(0)
	Invalid = State(-1)
)

func (r State) String() string {
	switch r {
	case Voice:
		return "Voice"
	case Noise:
		return "Noise"
	default:
		return "Error"
	}
}

type Mode C.int

const (
	Disable        = Mode(-1)
	Quality        = Mode(0)
	LowBitrate     = Mode(1)
	Aggressive     = Mode(2)
	VeryAggressive = Mode(3)
)

type VAD struct {
	ptr        *C.VadInst
	sampleRate int32
	mode       Mode
	mu         sync.Mutex
	closed     bool
}

func New(sampleRate int32, mode Mode) (*VAD, int32) {
	request := struct {
		inst   *C.VadInst
		mode   int32
		result int32
	}{
		mode:   int32(mode),
		result: 0,
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_vad_create), ptr, 0)
	if request.result != 0 {
		return nil, request.result
	}
	return &VAD{
		ptr:        request.inst,
		sampleRate: sampleRate,
		mode:       mode,
	}, 0
}

// Reset
// Re-initializes a VAD instance, clearing all state and resetting mode and
// sample rate to defaults.
func (v *VAD) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	request := struct {
		inst   *C.VadInst
		mode   int32
		result int32
	}{
		inst:   v.ptr,
		mode:   int32(v.mode),
		result: 0,
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_vad_reset), ptr, 0)
}

// SetMode
//
// Changes the VAD operating ("aggressiveness") mode of a VAD instance.
//
// A more aggressive (higher mode) VAD is more restrictive in reporting speech.
// Put in other words the probability of being speech when the VAD returns 1 is
// increased with increasing mode. As a consequence also the missed detection
// rate goes up.
//
// Valid modes are 0 ("quality"), 1 ("low bitrate"), 2 ("aggressive"), and 3
// ("very aggressive"). The default mode is 0.
//
// Returns 0 on success, or -1 if the specified mode is invalid.
func (v *VAD) SetMode(mode Mode) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return false
	}
	return C.WebRtcVad_set_mode(v.ptr, (C.int)(mode)) == 0
}

// SetSampleRate
//
// Sets the input sample rate in Hz for a VAD instance.
//
// Valid values are 8000, 16000, 32000 and 48000. The default is 8000. Note
// that internally all processing will be done 8000 Hz; input data in higher
// sample rates will just be downsampled first.
//
// Returns 0 on success, or -1 if the passed value is invalid.
func (v *VAD) SetSampleRate(sampleRate int32) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return false
	}
	v.sampleRate = sampleRate
	return true
	//return C.fvad_set_sample_rate(v.ptr, (C.int)(sampleRate)) == 0
}

// Process
//
// Calculates a VAD decision for an audio frame.
//
// `frame` is an array of `length` signed 16-bit samples. Only frames with a
// length of 10, 20 or 30 ms are supported, so for example at 8 kHz, `length`
// must be either 80, 160 or 240.
//
// Returns              : 1 - (active voice),
//                        0 - (non-active Voice),
//                       -1 - (invalid frame length).
func (v *VAD) Process(data []int16) State {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return Invalid
	}
	request := struct {
		ptr        *C.VadInst
		data       uintptr
		length     uintptr
		sampleRate int32
		result     int32
	}{
		ptr:        v.ptr,
		data:       (*reflect.SliceHeader)(unsafe.Pointer(&data)).Data,
		length:     uintptr(len(data)),
		sampleRate: v.sampleRate,
		result:     0,
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_vad_process), ptr, 0)
	return State(request.result)
}

func (v *VAD) ProcessNoLock(data []int16) State {
	if v.closed {
		return Invalid
	}
	request := struct {
		ptr        *C.VadInst
		data       uintptr
		length     uintptr
		sampleRate int32
		result     int32
	}{
		ptr:        v.ptr,
		data:       (*reflect.SliceHeader)(unsafe.Pointer(&data)).Data,
		length:     uintptr(len(data)),
		sampleRate: v.sampleRate,
		result:     0,
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_vad_process), ptr, 0)
	return State(request.result)
}

// Close and free up native resources.
func (v *VAD) Close() error {
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return nil
	}
	v.closed = true
	v.mu.Unlock()

	request := struct {
		ptr *C.VadInst
	}{
		ptr: v.ptr,
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_vad_free), ptr, 0)
	return nil
}

func DownSampleBy2(in []int16, out []int16, state *[8]int32) error {
	if len(out) != len(in)/2 {
		return errors.New("out must be have the size of in")
	}
	request := struct {
		in    uintptr
		size  uintptr
		out   uintptr
		state uintptr
	}{
		in:    (*reflect.SliceHeader)(unsafe.Pointer(&in)).Data,
		size:  uintptr(len(in)),
		out:   (*reflect.SliceHeader)(unsafe.Pointer(&out)).Data,
		state: uintptr(unsafe.Pointer(state)),
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_downsample_by_2), ptr, 0)
	return nil
}

func UpSampleBy2(in []int16, out []int16, state *[8]int32) error {
	if len(out) != len(in)*2 {
		return errors.New("out must be have the size of in")
	}
	request := struct {
		in    uintptr
		size  uintptr
		out   uintptr
		state uintptr
	}{
		in:    (*reflect.SliceHeader)(unsafe.Pointer(&in)).Data,
		size:  uintptr(len(in)),
		out:   (*reflect.SliceHeader)(unsafe.Pointer(&out)).Data,
		state: uintptr(unsafe.Pointer(state)),
	}
	ptr := uintptr(unsafe.Pointer(&request))
	cgo.NonBlocking((*byte)(C.do_webrtc_upsample_by_2), ptr, 0)
	return nil
}
