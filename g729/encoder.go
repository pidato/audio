package g729

/*
#cgo darwin,arm64 CXXFLAGS: -O2
#cgo darwin,amd64 LDFLAGS: -L./mac/amd64 -lbcg729
#cgo darwin,arm64 LDFLAGS: -L./mac/arm64 -lbcg729
#cgo linux,amd64 LDFLAGS: -L./linux/amd64 -lbcg729
#include "encoder.h"

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

//void bcg729Encoder(
//	bcg729EncoderChannelContextStruct *encoderChannelContext,
//	const int16_t inputFrame[],
//	uint8_t bitStream[],
//	uint8_t *bitStreamLength
//);

typedef struct do_initBcg729EncoderChannel_t {
	size_t ctx;
	uint8_t enableVAD;
} do_initBcg729EncoderChannel_t;

void do_initBcg729EncoderChannel(size_t arg0, size_t arg1) {
	do_initBcg729EncoderChannel_t* args = (do_initBcg729EncoderChannel_t*)(void*)arg0;
	args->ctx = (size_t)(void*)initBcg729EncoderChannel(args->enableVAD);
}

typedef struct do_closeBcg729EncoderChannel_t {
	size_t ctx;
} do_closeBcg729EncoderChannel_t;

void do_closeBcg729EncoderChannel(size_t arg0, size_t arg1) {
	do_closeBcg729EncoderChannel_t* args = (do_closeBcg729EncoderChannel_t*)(void*)arg0;
	closeBcg729EncoderChannel((bcg729EncoderChannelContextStruct *)(void*)args->ctx);
}

typedef struct do_bcg729Encoder_t {
	size_t ctx;
	size_t inputFrame;
	size_t bitStream;
	uint8_t bitStreamLength;
} do_bcg729Encoder_t;

void do_bcg729Encoder(size_t arg0, size_t arg1) {
	do_bcg729Encoder_t* args = (do_bcg729Encoder_t*)(void*)arg0;
	bcg729Encoder(
		(bcg729EncoderChannelContextStruct*)(void*)args->ctx,
		(const int16_t*)(void*)args->inputFrame,
		(uint8_t*)(void*)args->bitStream,
		&args->bitStreamLength
	);
}
*/
import "C"
import (
	"errors"
	"github.com/pidato/unsafe/cgo"
	"os"
	"unsafe"
)

type Encoder struct {
	enc uintptr
}

func NewEncoder(enableVAD bool) *Encoder {
	request := struct {
		ctx       uintptr
		enableVAD uint8
	}{
		ctx:       0,
		enableVAD: 0,
	}
	if enableVAD {
		request.enableVAD = 1
	}
	cgo.NonBlocking(
		(*byte)(C.do_initBcg729EncoderChannel),
		uintptr(unsafe.Pointer(&request)),
		0,
	)
	return &Encoder{enc: request.ctx}
}

func (e *Encoder) Close() error {
	enc := e.enc
	if enc == 0 {
		return os.ErrClosed
	}
	e.enc = 0
	request := struct {
		ctx uintptr
	}{
		ctx: enc,
	}
	cgo.NonBlocking(
		(*byte)(C.do_closeBcg729EncoderChannel),
		uintptr(unsafe.Pointer(&request)),
		0,
	)
	return nil
}

// Encode
//    parameters:
//      -(i) inputFrame : 80 samples (16 bits PCM)
//      -(o) bitStream : The 15 parameters for a frame on 80 bits
//           on 80 bits (5 16bits words) for voice frame, 4 on 2 byte for
//           noise frame, 0 for untransmitted frames
//      -(o) bitStreamLength : actual length of output, may be 0, 2 or 10
//           if VAD/DTX is enabled
func (e *Encoder) Encode(frame10ms []int16, encoded []byte) (uint8, error) {
	if len(frame10ms) < 80 {
		return 0, errors.New("frame10ms must be at least 80 samples")
	}
	if len(encoded) < 10 {
		return 0, errors.New("encoded size must be at least 10")
	}

	request := struct {
		ctx             uintptr
		inputFrame      uintptr
		bitStream       uintptr
		bitStreamLength uint8
	}{
		ctx:             e.enc,
		inputFrame:      uintptr(unsafe.Pointer(&frame10ms[0])),
		bitStream:       uintptr(unsafe.Pointer(&encoded[0])),
		bitStreamLength: uint8(10),
	}
	cgo.NonBlocking(
		(*byte)(C.do_bcg729Encoder),
		uintptr(unsafe.Pointer(&request)),
		0,
	)
	return request.bitStreamLength, nil
}
