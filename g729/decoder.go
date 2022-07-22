package g729

/*
#cgo darwin,amd64 LDFLAGS: -L./mac/amd64 -lbcg729
#cgo darwin,arm64 LDFLAGS: -L./mac/arm64 -lbcg729
#cgo linux,amd64 LDFLAGS: -L./linux/amd64 -lbcg729
#include "decoder.h"
#include "typedef.h"

///////////////////////////////////////////////////////////////////////////////
// parametersArray2BitStream : convert array of parameters to bitStream      //
//      according to spec 4 - Table 8 and following mapping of values        //
//               0 -> L0 (1 bit)                                             //
//               1 -> L1 (7 bits)                                            //
//               2 -> L2 (5 bits)                                            //
//               3 -> L3 (5 bits)                                            //
//               4 -> P1 (8 bit)                                             //
//               5 -> P0 (1 bits)                                            //
//               6 -> C1 (13 bits)                                           //
//               7 -> S1 (4 bits)                                            //
//               8 -> GA1(3 bits)                                            //
//               9 -> GB1(4 bits)                                            //
//               10 -> P2 (5 bits)                                           //
//               11 -> C2 (13 bits)                                          //
//               12 -> S2 (4 bits)                                           //
//               13 -> GA2(3 bits)                                           //
//               14 -> GB2(4 bits)                                           //
//    parameters:                                                            //
//      -(i) parameters : 16 values parameters array                         //
//      -(o) bitStream : the 16 values streamed on 80 bits in a              //
//           10/8bits values array                                           //
//                                                                           //
///////////////////////////////////////////////////////////////////////////////
void parametersArray2BitStream(uint16_t parameters[], uint8_t bitStream[]);
//void parametersArray2BitStream(uint16_t parameters[], uint8_t bitStream[])
//{
//	bitStream[0] = ((parameters[0]&((uint16_t) 0x1))<<7) |
//	(parameters[1]&((uint16_t) 0x7f));
//
//	bitStream[1] = ((parameters[2]&((uint16_t) 0x1f))<<3) |
//	((parameters[3]>>2)&((uint16_t) 0x7));
//
//	bitStream[2] = ((parameters[3]&((uint16_t) 0x3))<<6) |
//	((parameters[4]>>2)&((uint16_t) 0x3f));
//
//	bitStream[3] = ((parameters[4]&((uint16_t) 0x3))<<6) |
//	((parameters[5]&((uint16_t) 0x1))<<5) |
//	((parameters[6]>>8)&((uint16_t) 0x1f));
//
//	bitStream[4] = ((parameters[6])&((uint16_t) 0xff));
//
//	bitStream[5] = ((parameters[7]&((uint16_t) 0xf))<<4) |
//	((parameters[8]&((uint16_t) 0x7))<<1) |
//	((parameters[9]>>3)&((uint16_t) 0x1));
//
//	bitStream[6] = ((parameters[9]&((uint16_t) 0x7))<<5) |
//	(parameters[10]&((uint16_t) 0x1f));
//
//	bitStream[7] = ((parameters[11]>>5)&((uint16_t) 0xff));
//
//	bitStream[8] = ((parameters[11]&((uint16_t) 0x1f))<<3) |
//	((parameters[12]>>1)&((uint16_t) 0x7));
//
//	bitStream[9] = ((parameters[12]&((uint16_t) 0x1))<<7) |
//	((parameters[13]&((uint16_t) 0x7))<<4) |
//	(parameters[14]&((uint16_t) 0xf));
//}

///////////////////////////////////////////////////////////////////////////////
// parametersArray2BitStream : convert bitStream to an array of parameters   //
//             reverse operation of previous funtion                         //
//    parameters:                                                            //
//      -(i) bitStream : the 16 values streamed on 80 bits in a              //
//           10/8bits values array                                           //
//      -(o) parameters : 16 values parameters array                         //
//                                                                           //
///////////////////////////////////////////////////////////////////////////////
void parametersBitStream2Array(uint8_t bitStream[], uint16_t parameters[]);
//void parametersBitStream2Array(uint8_t bitStream[], uint16_t parameters[])
//{
//	parameters[0] = (bitStream[0]>>7)&(uint16_t)0x1;
//	parameters[1] = bitStream[0]&(uint16_t)0x7f;
//	parameters[2] = (bitStream[1]>>3)&(uint16_t)0x1f;
//	parameters[3] = (((uint16_t)bitStream[1]&(uint16_t)0x7)<<2) | ((bitStream[2]>>6)&(uint16_t)0x3);
//	parameters[4] = (((uint16_t)bitStream[2])&(uint16_t)0x3f)<<2 | ((bitStream[3]>>6)&(uint16_t)0x3);;
//	parameters[5] = (bitStream[3]>>5)&(uint16_t)0x1;
//	parameters[6] = (((uint16_t)(bitStream[3]&(uint16_t)0x1f))<<8)| bitStream[4];
//	parameters[7] = (bitStream[5]>>4)&(uint16_t)0xf;
//	parameters[8] = (bitStream[5]>>1)&(uint16_t)0x7;
//	parameters[9] = (((uint16_t)bitStream[5]&(uint16_t)0x1)<<3)|((bitStream[6]>>5)&(uint16_t)0x7);
//	parameters[10]= (uint16_t)bitStream[6]&(uint16_t)0x1f;
//	parameters[11]= (((uint16_t)bitStream[7])<<5)|((bitStream[8]>>3)&(uint16_t)0x1f);
//	parameters[12]= ((bitStream[8]&(uint16_t)0x7)<<1) | ((bitStream[9]>>7)&(uint16_t)0x1);
//	parameters[13]= (bitStream[9]>>4)&(uint16_t)0x7;
//	parameters[14]= bitStream[9]&(uint16_t)0xf;
//}
//void bcg729Decoder(
//	bcg729DecoderChannelContextStruct *decoderChannelContext,
//	const uint8_t bitStream[],
//	uint8_t bitStreamLength,
//	uint8_t frameErasureFlag,
//	uint8_t SIDFrameFlag,
//	uint8_t rfc3389PayloadFlag,
//	int16_t signal[]
//);

typedef struct do_initBcg729DecoderChannel_t {
	size_t ctx;
} do_initBcg729DecoderChannel_t;

void do_initBcg729DecoderChannel(size_t arg0, size_t arg1) {
	do_initBcg729DecoderChannel_t* args = (do_initBcg729DecoderChannel_t*)(void*)arg0;
	args->ctx = (size_t)(void*)initBcg729DecoderChannel();
}

typedef struct do_closeBcg729DecoderChannel_t {
	size_t ctx;
} do_closeBcg729DecoderChannel_t;

void do_closeBcg729DecoderChannel(size_t arg0, size_t arg1) {
	do_closeBcg729DecoderChannel_t* args = (do_closeBcg729DecoderChannel_t*)(void*)arg0;
	closeBcg729DecoderChannel((bcg729DecoderChannelContextStruct *)(void*)args->ctx);
}

typedef struct do_bcg729Decoder_t {
	size_t ctx;
	size_t bitStream;
	size_t signal;
	uint8_t bitStreamLength;
	uint8_t frameErasureFlag;
	uint8_t SIDFrameFlag;
	uint8_t rfc3389PayloadFlag;
	//uint8_t previousFrameIsActiveFlag;
} do_bcg729Decoder_t;

void do_bcg729Decoder(size_t arg0, size_t arg1) {
	do_bcg729Decoder_t* args = (do_bcg729Decoder_t*)(void*)arg0;
	bcg729Decoder(
		(bcg729DecoderChannelContextStruct*)(void*)args->ctx,
		(const uint8_t*)(void*)args->bitStream,
		args->bitStreamLength,
		args->frameErasureFlag,
		args->SIDFrameFlag,
		args->rfc3389PayloadFlag,
		(int16_t*)(void*)args->signal
	);
	//args->previousFrameIsActiveFlag = ((bcg729DecoderChannelContextStruct*)(void*)args->ctx)-> previousFrameIsActiveFlag;
}
*/
import "C"
import (
	"errors"
	"github.com/pidato/unsafe/cgo"
	"os"
	"unsafe"
)

type Decoder struct {
	dec     uintptr
	isVoice bool
}

func NewDecoder() *Decoder {
	request := struct {
		ctx uintptr
	}{
		ctx: 0,
	}
	cgo.NonBlocking(
		(*byte)(C.do_initBcg729DecoderChannel),
		uintptr(unsafe.Pointer(&request)),
		0,
	)
	return &Decoder{dec: request.ctx}
}

func (d *Decoder) IsVoice() bool { return d.isVoice }

func (d *Decoder) Close() error {
	dec := d.dec
	if dec == 0 {
		return os.ErrClosed
	}
	d.dec = 0
	request := struct {
		ctx uintptr
	}{
		ctx: dec,
	}
	cgo.NonBlocking(
		(*byte)(C.do_closeBcg729DecoderChannel),
		uintptr(unsafe.Pointer(&request)),
		0,
	)
	return nil
}

// Decode
//    parameters:
//      -(i) bitStream : 15 parameters on 80 bits
//      -(o) signal : a decoded frame 80 samples (16 bits PCM)
func (d *Decoder) Decode(
	packet []byte,
	decoded []int16,
) error {
	if len(decoded) < 80 {
		return errors.New("decoded must have a length of 80")
	}
	request := struct {
		ctx                uintptr
		bitStream          uintptr
		signal             uintptr
		bitStreamLength    uint8
		frameErasureFlag   uint8
		SIDFrameFlag       uint8
		rfc3389PayloadFlag uint8
		//previousFrameIsActiveFlag uint8
	}{
		ctx:                d.dec,
		signal:             uintptr(unsafe.Pointer(&decoded[0])),
		bitStreamLength:    uint8(len(packet)),
		frameErasureFlag:   0,
		SIDFrameFlag:       0,
		rfc3389PayloadFlag: 0,
		//previousFrameIsActiveFlag: 0,
	}
	if len(packet) > 0 {
		request.bitStream = uintptr(unsafe.Pointer(&packet[0]))
	}
	cgo.NonBlocking(
		(*byte)(C.do_bcg729Decoder),
		uintptr(unsafe.Pointer(&request)),
		0,
	)

	//if request.previousFrameIsActiveFlag == 0 {
	//	d.isVoice = false
	//} else {
	//	d.isVoice = true
	//}

	return nil
}

// DecodeWithOptions
//    parameters:
//      -(i) bitStream : 15 parameters on 80 bits
//      -(i): bitStreamLength : in bytes, length of previous buffer
//      -(i) frameErased: flag: true, frame has been erased
//      -(i) SIDFrameFlag: flag: true, frame is a SID one
//      -(i) rfc3389PayloadFlag: true when CN payload follow rfc3389
//      -(o) signal : a decoded frame 80 samples (16 bits PCM)
func (d *Decoder) DecodeWithOptions(
	packet []byte,
	frameErased, SIDFrameFlag, rfc3389PayloadFlag bool,
	decoded []int16,
) error {
	if len(decoded) < 80 {
		return errors.New("decoded must have a length of 80")
	}
	var frameErased_ uint8
	if frameErased {
		frameErased_ = 1
	} else {
		frameErased_ = 0
	}
	var SIDFrameFlag_ uint8
	if SIDFrameFlag {
		SIDFrameFlag_ = 1
	} else {
		SIDFrameFlag_ = 0
	}
	var rfc3389PayloadFlag_ uint8
	if rfc3389PayloadFlag {
		rfc3389PayloadFlag_ = 1
	} else {
		rfc3389PayloadFlag_ = 0
	}
	request := struct {
		ctx                uintptr
		bitStream          uintptr
		signal             uintptr
		bitStreamLength    uint8
		frameErasureFlag   uint8
		SIDFrameFlag       uint8
		rfc3389PayloadFlag uint8
		//previousFrameIsActiveFlag uint8
	}{
		ctx:                d.dec,
		signal:             uintptr(unsafe.Pointer(&decoded[0])),
		bitStreamLength:    uint8(len(packet)),
		frameErasureFlag:   frameErased_,
		SIDFrameFlag:       SIDFrameFlag_,
		rfc3389PayloadFlag: rfc3389PayloadFlag_,
		//previousFrameIsActiveFlag: 0,
	}
	if len(packet) > 0 {
		request.bitStream = uintptr(unsafe.Pointer(&packet[0]))
	}
	cgo.NonBlocking(
		(*byte)(C.do_bcg729Decoder),
		uintptr(unsafe.Pointer(&request)),
		0,
	)

	//if request.previousFrameIsActiveFlag == 0 {
	//	d.isVoice = false
	//} else {
	//	d.isVoice = true
	//}

	//C.bcg729Decoder(d.dec,
	//	(*C.uint8_t)(bitStream),
	//	C.uint8_t(len(packet)),
	//	frameErased_,
	//	SIDFrameFlag_,
	//	rfc3389PayloadFlag_,
	//	(*C.int16_t)(signal))
	return nil
}
