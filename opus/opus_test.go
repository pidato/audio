// Copyright © 2015-2017 Go Opus Authors (see AUTHORS file)
//
// License for use of this code is detailed in the LICENSE file

package opus

import (
	"encoding/binary"
	"fmt"
	"github.com/go-audio/wav"
	"github.com/pidato/audio/g729"
	"github.com/pidato/audio/pcm"
	//"github.com/pion/webrtc/v2/pkg/media/oggwriter"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestVersion(t *testing.T) {
	if ver := Version(); !strings.HasPrefix(ver, "libopus") {
		t.Errorf("Unexpected linked libopus Version: " + ver)
	}
}

func TestOpusErrstr(t *testing.T) {
	// I scooped this -1 up from opus_defines.h, it's OPUS_BAD_ARG. Not pretty,
	// but it's better than not testing at all. Again, accessing #defines from
	// CGO is not possible.
	if ErrBadArg.Error() != "opus: invalid argument" {
		t.Errorf("Expected \"invalid argument\" error message for error code %d: %v",
			ErrBadArg, ErrBadArg)
	}
}

func TestCodec(t *testing.T) {
	// Create bogus input sound
	const G4 = 391.995
	const SAMPLE_RATE = 16000
	const FRAME_SIZE_MS = 10
	const FRAME_SIZE = SAMPLE_RATE * FRAME_SIZE_MS / 1000
	pcm := make([]int16, FRAME_SIZE)
	enc, err := NewEncoder(SAMPLE_RATE, 1, AppVoIP)
	if err != nil || enc == nil {
		t.Fatalf("Error creating new encoder: %v", err)
	}
	addSine(pcm, SAMPLE_RATE, G4)
	data := make([]byte, 1000)
	n, err := enc.Encode(pcm, data)
	if err != nil {
		t.Fatalf("Couldn't encode data: %v", err)
	}
	sampleRate, err := enc.SampleRate()
	if err != nil {
		t.Fatal(err)
	}
	bitrate, err := enc.Bitrate()
	if err != nil {
		t.Fatal(err)
	}
	_ = bitrate
	_ = sampleRate
	data = data[:n]
	dec, err := NewDecoder(SAMPLE_RATE, 1)
	if err != nil || dec == nil {
		t.Fatalf("Error creating new decoder: %v", err)
	}
	n, err = dec.Decode(data, pcm)
	if err != nil {
		t.Fatalf("Couldn't decode data: %v", err)
	}
	if len(pcm) != n {
		t.Fatalf("Length mismatch: %d samples in, %d out", len(pcm), n)
	}
	// Checking the output programmatically is seriously not easy. I checked it
	// by hand and by ear, it looks fine. I'll just assume the API faithfully
	// passes error codes through, and that's that.
}

func getPCM(t *testing.T, filename string) []byte {
	file, err := os.Open("testdata/recording.pcm")
	if err != nil {
		t.Fatal(err)
	}
	decoder := wav.NewDecoder(file)
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		t.Fatal(err)
	}
	bytes := make([]byte, len(buf.Data)*2)
	for i, sample := range buf.Data {
		binary.LittleEndian.PutUint16(bytes[i:i+2], uint16(sample))
	}
	return bytes
}

func toOpus(t *testing.T, filename string, ptime int) (frames [][]byte, sampleRate int, frameSize int) {
	ptime = 10
	file, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := pcm.OpenWav(file, ptime)
	if err != nil {
		t.Fatal(err)
	}

	sampleRate = reader.SampleRate()
	sampleRateAdjustment := float32(48000) / float32(sampleRate)
	opusFrameRate := int(float32(reader.FrameSize()) * sampleRateAdjustment)
	_ = opusFrameRate

	frameSize = reader.FrameSize()
	_ = frameSize
	//frameSize := int((int64(ptime) * int64(time.Millisecond)) / int64(sampleDuration))

	enc, err := NewEncoder(sampleRate, 1, AppVoIP)
	if err != nil || enc == nil {
		t.Fatalf("Error creating new encoder: %v", err)
	}

	opusSampleRate, err := enc.SampleRate()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(opusSampleRate)

	err = enc.SetComplexity(10)
	if err != nil {
		t.Fatal(err)
	}
	err = enc.SetMaxBandwidth(Narrowband)
	if err != nil {
		t.Fatal(err)
	}
	err = enc.SetBitrate(8000)
	if err != nil {
		t.Fatal(err)
	}
	//err = enc.SetInBandFEC(true)
	//if err != nil {
	//	t.Fatal(err)
	//}

	//packetizer := rtp.NewPacketizerOpus(
	//	1200,
	//	webrtc.DefaultPayloadTypeOpus,
	//	rand.Uint32(),
	//	&codecs.OpusPayloader{},
	//	rtp.NewFixedSequencer(0),
	//	48000,
	//)
	//_ = packetizer

	g729File, err := os.OpenFile("testdata/vad.g729", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0755)
	defer g729File.Close()
	g729Enc := g729.NewEncoder(true)
	buf := make([]byte, 10)
	opusWriter, err := CreateFile("testdata/1.opus", uint32(sampleRate), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	opusFrames := make([][]byte, 0, 128)

	eof := false
	for !eof {
		frame, err := reader.ReadFrame()
		if err != nil {
			eof = err == io.EOF
			if !eof {
				break
			}
		}
		if len(frame) == 0 {
			err = opusWriter.WriteEOF(nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if len(frame) != frameSize {
			l := len(frame)
			frame = frame[0:frameSize]
			for i := 0; i < l; i++ {
				frame[i] = 0
			}
		}
		bitstreamLength, err := g729Enc.Encode(frame, buf)
		if err != nil {
			t.Fatalf("Error creating new encoder: %v", err)
		}

		if bitstreamLength > 0 {
			_, _ = g729File.Write(buf[0:bitstreamLength])
		}
		opus := make([]byte, frameSize*2)
		n, err := enc.Encode(frame, opus)
		if err != nil {
			t.Fatalf("Error creating new encoder: %v", err)
		}

		opusFrame := opus[:n]
		opusFrames = append(opusFrames, opusFrame)

		//packets := packetizer.Packetize(opusFrame, uint32(len(frame)))
		//packets := packetizer.Packetize(opusFrame, uint32(opusFrameRate))
		//for _, packet := range packets {
		//	err = opusWriter.WriteRTP(packet)
		//	if err != nil {
		//		t.Fatal(err)
		//	}
		//}

		if eof {
			err = opusWriter.WriteEOF(opusFrame, opusFrameRate)
		} else {
			err = opusWriter.Write(opusFrame, opusFrameRate)
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	err = opusWriter.Close()
	if err != nil {
		t.Fatal(err)
	}
	return opusFrames, sampleRate, frameSize
}

func TestWavReaderInternal(t *testing.T) {
	reader, _ := pcm.OpenWavFile("testdata/recording.pcm", 10)

	outFile, _ := os.OpenFile("testdata/recording.raw", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)

	//buf := make([]byte, 320)
	for {
		frame, err := reader.ReadFrame()
		if len(frame) > 0 {
			pcmBytes := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
				Data: (uintptr)(unsafe.Pointer(&frame[0])),
				Len:  len(frame) * 2,
				Cap:  len(frame) * 2,
			}))
			outFile.Write(pcmBytes)
		}
		if err != nil {
			break
		}

		//samples, err := reader.ReadSamples()
		//if err == io.EOF {
		//	break
		//}
		//
		//_
		//for _, sample := range samples {
		//	fmt.Printf("L/R: %d/%d\n", reader.IntValue(sample, 0), reader.IntValue(sample, 1))
		//}
	}

	outFile.Close()
}

func TestOpusWriteFile(t *testing.T) {
	//opusFrames, sampleRate, frameSize := toOpus(t, "testdata/recording.wav", 20)
	opusFrames, sampleRate, frameSize := toOpus(t, "testdata/8khz_sample_000000.wav", 20)
	//opusFrames, SampleRate, frameSize := toOpus(t, "testdata/speech_8.pcm", 10)

	_ = sampleRate
	//SampleRate = 48000
	opusSampleRate := 16000
	dec, err := NewDecoder(opusSampleRate, 1)
	if err != nil {
		t.Fatal(err)
	}

	rawPCM, err := os.OpenFile("testdata/rec.pcm", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		t.Fatal(err)
	}

	sampleCount := 0
	for _, frame := range opusFrames {
		//_, _ = opusOut.Write(frame)
		pcmFrame := make([]int16, frameSize*10)
		n, err := dec.Decode(frame, pcmFrame)
		if err != nil {
			t.Fatal(err)
		}
		if len(pcmFrame) != n {
			pcmFrame = pcmFrame[:n]
		}
		sampleCount += len(pcmFrame)

		pcmBytes := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: (uintptr)(unsafe.Pointer(&pcmFrame[0])),
			Len:  len(pcmFrame) * 2,
			Cap:  len(pcmFrame) * 2,
		}))

		_, _ = rawPCM.Write(pcmBytes)
	}

	_ = rawPCM.Close()
}

//func TestCodecWav(t *testing.T) {
//	file, err := os.Open("testdata/recording.pcm")
//	if err != nil {
//		t.Fatal(err)
//	}
//	reader, err := pcm.OpenWav(file, 10)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	var SAMPLE_RATE = reader.SampleRate()
//	var FRAME_SIZE_MS = 10
//	var FRAME_SIZE = SAMPLE_RATE * FRAME_SIZE_MS / 1000
//
//	enc, err := NewEncoder(SAMPLE_RATE, 1, AppVoIP)
//	if err != nil || enc == nil {
//		t.Fatalf("Error creating new encoder: %v", err)
//	}
//
//	encodedData := make([][]byte, 0, 128)
//
//	output := bytes.NewBuffer(nil)
//	eof := false
//	for !eof {
//		frame, err := reader.ReadFrame()
//		if err != nil {
//			eof = err == io.EOF
//			if !eof {
//				break
//			}
//		}
//
//		if len(frame) == 0 {
//			break
//		}
//
//		if len(frame) != FRAME_SIZE {
//			l := len(frame)
//			frame = frame[0:FRAME_SIZE]
//			for i := 0; i < l; i++ {
//				frame[i] = 0
//			}
//		}
//
//		data := make([]byte, FRAME_SIZE*3)
//		n, err := enc.Encode(frame, data)
//		if err != nil {
//			t.Fatal(err)
//		}
//		if n <= 0 {
//			continue
//		}
//		data = data[:n]
//		encodedData = append(encodedData, data)
//
//		wn, err := output.Write(data)
//		if wn != n {
//			t.Fatal(fmt.Errorf(""))
//		}
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	sampleRate, err := enc.SampleRate()
//	if err != nil {
//		t.Fatal(err)
//	}
//	bitrate, err := enc.Bitrate()
//	if err != nil {
//		t.Fatal(err)
//	}
//	_ = bitrate
//	_ = sampleRate
//
//	//outfile, err := os.OpenFile("testdata/recording.ogg", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
//	//if err != nil {
//	//	t.Fatal(err)
//	//}
//
//	//writer, err := oggwriter.CreateFile("recording.opus", 16000, 1)
//	//writer := ogg.NewEncoder(0, outfile)
//	ogwriter, err := CreateFile("testdata/recording.opus", uint32(sampleRate), uint16(enc.channels))
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	//packetizer := rtp.NewPacketizer(1200, webrtc.DefaultPayloadTypeOpus, 100, &codecs.OpusPayloader{}, rtp.NewRandomSequencer(), uint32(SampleRate))
//	//
//	//frameSize := uint32(FRAME_SIZE)
//	//
//
//	//packets := packetizer.Packetize(output.Bytes(), frameSize)
//	//for _, packet := range packets {
//	//	err = ogwriter.WriteRTP(packet)
//	//	if err != nil {
//	//		t.Fatal(err)
//	//	}
//	//}
//
//	//for _, frame := range encodedData {
//	//	packets := packetizer.Packetize(frame, frameSize)
//	//	for _, packet := range packets {
//	//		err = ogwriter.WriteRTP(packet)
//	//		if err != nil {
//	//			t.Fatal(err)
//	//		}
//	//	}
//	//}
//	err = ogwriter.Close()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	//for granule, frame := range encodedData {
//	//	if granule == 0 {
//	//		err = writer.EncodeBOS(int64(granule), frame)
//	//		if err != nil {
//	//			t.Fatal(err)
//	//		}
//	//	} else {
//	//		err = writer.Encode(int64(granule), frame)
//	//		if err != nil {
//	//			t.Fatal(err)
//	//		}
//	//	}
//	//}
//	//writer.EncodeEOS()
//	//outfile.Write(output.Bytes())
//
//	_pcm := make([]int16, FRAME_SIZE)
//	dec, err := NewDecoder(SAMPLE_RATE, 1)
//	if err != nil || dec == nil {
//		t.Fatalf("Error creating new decoder: %v", err)
//	}
//
//	file, err = os.Open("testdata/recording.pcm")
//	if err != nil {
//		t.Fatal(err)
//	}
//	reader, err = pcm.OpenWav(file, 10)
//	readerFile, err := os.Open("testdata/recording.pcm")
//
//	wdec := wav.NewDecoder(readerFile)
//	fullpcm, err := wdec.FullPCMBuffer()
//	println(fullpcm)
//
//	outfile, err := os.OpenFile("testdata/rec.pcm", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
//	wavencoder := wav.NewEncoder(outfile, int(wdec.SampleRate), int(wdec.SampleBitDepth()), int(wdec.NumChans), int(wdec.WavAudioFormat))
//	//wavencoder := pcm.NewEncoder(outfile, 16000, 16, 1, int(reader.WavAudioFormat()))
//
//
//	for {
//		ff, err := reader.ReadFrame()
//		if err != nil {
//			break
//		}
//		err = wavencoder.WriteFrame(ff)
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//	//for _, frame := range encodedData {
//	//
//	//	n, err := dec.DecodeWithOptions(frame, pcm)
//	//	if err != nil {
//	//		t.Fatalf("Couldn't decode data: %v", err)
//	//	}
//	//	if len(pcm) != n {
//	//		t.Fatalf("Length mismatch: %d samples in, %d out", len(pcm), n)
//	//	}
//	//	wavencoder.WriteFrame(pcm)
//	//}
//
//	wavencoder.Close()
//	outfile.Close()
//
//	// Checking the output programmatically is seriously not easy. I checked it
//	// by hand and by ear, it looks fine. I'll just assume the API faithfully
//	// passes error codes through, and that's that.
//}

func TestCodecFloat32(t *testing.T) {
	// Create bogus input sound
	const G4 = 391.995
	const SAMPLE_RATE = 48000
	const FRAME_SIZE_MS = 60
	const FRAME_SIZE = SAMPLE_RATE * FRAME_SIZE_MS / 1000
	pcm := make([]float32, FRAME_SIZE)
	enc, err := NewEncoder(SAMPLE_RATE, 1, AppVoIP)
	if err != nil || enc == nil {
		t.Fatalf("Error creating new encoder: %v", err)
	}
	addSineFloat32(pcm, SAMPLE_RATE, G4)
	data := make([]byte, 1000)
	n, err := enc.EncodeFloat32(pcm, data)
	if err != nil {
		t.Fatalf("Couldn't encode data: %v", err)
	}
	dec, err := NewDecoder(SAMPLE_RATE, 1)
	if err != nil || dec == nil {
		t.Fatalf("Error creating new decoder: %v", err)
	}
	// TODO: Uh-oh.. it looks like I forgot to put a data = data[:n] here, yet
	// the test is not failing. Why?
	n, err = dec.DecodeFloat32(data, pcm)
	if err != nil {
		t.Fatalf("Couldn't decode data: %v", err)
	}
	if len(pcm) != n {
		t.Fatalf("Length mismatch: %d samples in, %d out", len(pcm), n)
	}
}

func TestCodecFEC(t *testing.T) {
	// Create bogus input sound
	const G4 = 391.995
	const SAMPLE_RATE = 48000
	const FRAME_SIZE_MS = 10
	const FRAME_SIZE = SAMPLE_RATE * FRAME_SIZE_MS / 1000
	const NUMBER_OF_FRAMES = 6
	pcm := make([]int16, 0)

	enc, err := NewEncoder(SAMPLE_RATE, 1, AppVoIP)
	if err != nil || enc == nil {
		t.Fatalf("Error creating new encoder: %v", err)
	}
	enc.SetPacketLossPerc(30)
	enc.SetInBandFEC(true)

	dec, err := NewDecoder(SAMPLE_RATE, 1)
	if err != nil || dec == nil {
		t.Fatalf("Error creating new decoder: %v", err)
	}

	mono := make([]int16, FRAME_SIZE*NUMBER_OF_FRAMES)
	addSine(mono, SAMPLE_RATE, G4)

	encodedData := make([][]byte, NUMBER_OF_FRAMES)
	for i, idx := 0, 0; i < len(mono); i += FRAME_SIZE {
		data := make([]byte, 1000)
		n, err := enc.Encode(mono[i:i+FRAME_SIZE], data)
		if err != nil {
			t.Fatalf("Couldn't encode data: %v", err)
		}
		data = data[:n]
		encodedData[idx] = data
		idx++
	}

	lost := false
	for i := 0; i < len(encodedData); i++ {
		// "Dropping" packets 2 and 4
		if i == 2 || i == 4 {
			lost = true
			continue
		}
		if lost {
			samples, err := dec.LastPacketDuration()
			if err != nil {
				t.Fatalf("Couldn't get last packet duration: %v", err)
			}

			pcmBuffer := make([]int16, samples)
			if err = dec.DecodeFEC(encodedData[i], pcmBuffer); err != nil {
				t.Fatalf("Couldn't decode data: %v", err)
			}
			pcm = append(pcm, pcmBuffer...)
			lost = false
		}

		pcmBuffer := make([]int16, NUMBER_OF_FRAMES*FRAME_SIZE)
		n, err := dec.Decode(encodedData[i], pcmBuffer)
		if err != nil {
			t.Fatalf("Couldn't decode data: %v", err)
		}
		pcmBuffer = pcmBuffer[:n]
		if n != FRAME_SIZE {
			t.Fatalf("Length mismatch: %d samples expected, %d out", FRAME_SIZE, n)
		}
		pcm = append(pcm, pcmBuffer...)
	}

	if len(mono) != len(pcm) {
		t.Fatalf("Input/Output length mismatch: %d samples in, %d out", len(mono), len(pcm))
	}

	// Commented out for the same reason as in TestStereo
	/*
		fmt.Printf("in,out\n")
		for i := range mono {
			fmt.Printf("%d,%d\n", mono[i], pcm[i])
		}
	*/

}

func TestCodecFECFloat32(t *testing.T) {
	// Create bogus input sound
	const G4 = 391.995
	const SAMPLE_RATE = 48000
	const FRAME_SIZE_MS = 10
	const FRAME_SIZE = SAMPLE_RATE * FRAME_SIZE_MS / 1000
	const NUMBER_OF_FRAMES = 6
	pcm := make([]float32, 0)

	enc, err := NewEncoder(SAMPLE_RATE, 1, AppVoIP)
	if err != nil || enc == nil {
		t.Fatalf("Error creating new encoder: %v", err)
	}
	enc.SetPacketLossPerc(30)
	enc.SetInBandFEC(true)

	dec, err := NewDecoder(SAMPLE_RATE, 1)
	if err != nil || dec == nil {
		t.Fatalf("Error creating new decoder: %v", err)
	}

	mono := make([]float32, FRAME_SIZE*NUMBER_OF_FRAMES)
	addSineFloat32(mono, SAMPLE_RATE, G4)

	encodedData := make([][]byte, NUMBER_OF_FRAMES)
	for i, idx := 0, 0; i < len(mono); i += FRAME_SIZE {
		data := make([]byte, 1000)
		n, err := enc.EncodeFloat32(mono[i:i+FRAME_SIZE], data)
		if err != nil {
			t.Fatalf("Couldn't encode data: %v", err)
		}
		data = data[:n]
		encodedData[idx] = data
		idx++
	}

	lost := false
	for i := 0; i < len(encodedData); i++ {
		// "Dropping" packets 2 and 4
		if i == 2 || i == 4 {
			lost = true
			continue
		}
		if lost {
			samples, err := dec.LastPacketDuration()
			if err != nil {
				t.Fatalf("Couldn't get last packet duration: %v", err)
			}

			pcmBuffer := make([]float32, samples)
			if err = dec.DecodeFECFloat32(encodedData[i], pcmBuffer); err != nil {
				t.Fatalf("Couldn't decode data: %v", err)
			}
			pcm = append(pcm, pcmBuffer...)
			lost = false
		}

		pcmBuffer := make([]float32, NUMBER_OF_FRAMES*FRAME_SIZE)
		n, err := dec.DecodeFloat32(encodedData[i], pcmBuffer)
		if err != nil {
			t.Fatalf("Couldn't decode data: %v", err)
		}
		pcmBuffer = pcmBuffer[:n]
		if n != FRAME_SIZE {
			t.Fatalf("Length mismatch: %d samples expected, %d out", FRAME_SIZE, n)
		}
		pcm = append(pcm, pcmBuffer...)
	}

	if len(mono) != len(pcm) {
		t.Fatalf("Input/Output length mismatch: %d samples in, %d out", len(mono), len(pcm))
	}

	// Commented out for the same reason as in TestStereo
	/*
		fmt.Printf("in,out\n")
		for i := 0; i < len(mono); i++ {
			fmt.Printf("%f,%f\n", mono[i], pcm[i])
		}
	*/
}

func TestStereo(t *testing.T) {
	// Create bogus input sound
	const G4 = 391.995
	const E3 = 164.814
	const SAMPLE_RATE = 48000
	const FRAME_SIZE_MS = 60
	const CHANNELS = 2
	const FRAME_SIZE_MONO = SAMPLE_RATE * FRAME_SIZE_MS / 1000

	enc, err := NewEncoder(SAMPLE_RATE, CHANNELS, AppVoIP)
	if err != nil || enc == nil {
		t.Fatalf("Error creating new encoder: %v", err)
	}
	dec, err := NewDecoder(SAMPLE_RATE, CHANNELS)
	if err != nil || dec == nil {
		t.Fatalf("Error creating new decoder: %v", err)
	}

	// Source signal (left G4, right E3)
	left := make([]int16, FRAME_SIZE_MONO)
	right := make([]int16, FRAME_SIZE_MONO)
	addSine(left, SAMPLE_RATE, G4)
	addSine(right, SAMPLE_RATE, E3)
	pcm := interleave(left, right)

	data := make([]byte, 1000)
	n, err := enc.Encode(pcm, data)
	if err != nil {
		t.Fatalf("Couldn't encode data: %v", err)
	}
	data = data[:n]
	n, err = dec.Decode(data, pcm)
	if err != nil {
		t.Fatalf("Couldn't decode data: %v", err)
	}
	if n*CHANNELS != len(pcm) {
		t.Fatalf("Length mismatch: %d samples in, %d out", len(pcm), n*CHANNELS)
	}

	// This is hard to check programatically, but I plotted the graphs in a
	// spreadsheet and it looked great. The encoded waves both start out with a
	// string of zero samples before they pick up, and the G4 is phase shifted
	// by half a period, but that's all fine for lossy audio encoding.
	/*
		leftdec, rightdec := split(pcm)
		fmt.Printf("left_in,left_out,right_in,right_out\n")
		for i := range left {
			fmt.Printf("%d,%d,%d,%d\n", left[i], leftdec[i], right[i], rightdec[i])
		}
	*/
}

func TestBufferSize(t *testing.T) {
	const G4 = 391.995
	const SAMPLE_RATE = 48000
	const FRAME_SIZE_MS = 60
	const FRAME_SIZE = SAMPLE_RATE * FRAME_SIZE_MS / 1000
	const GUARD_SIZE = 100

	checkGuardInt16 := func(t *testing.T, s []int16) {
		for n := range s {
			if s[n] != 0 {
				t.Fatal("Memory corruption detected")
			}
		}
	}

	checkGuardFloat32 := func(t *testing.T, s []float32) {
		for n := range s {
			if s[n] != 0 {
				t.Fatal("Memory corruption detected")
			}
		}
	}

	checkResult := func(t *testing.T, n int, err error, expectOK bool) {
		if expectOK {
			if err != nil {
				t.Fatalf("Couldn't decode data: %v", err)
			}
			if n != FRAME_SIZE {
				t.Fatalf("Length mismatch: %d samples in, %d out", FRAME_SIZE, n)
			}
		} else {
			if err == nil {
				t.Fatalf("Expected decode failure, but it succeeded")
			}
		}
	}

	encodeFrame := func(t *testing.T) []byte {
		pcm := make([]int16, FRAME_SIZE)
		enc, err := NewEncoder(SAMPLE_RATE, 1, AppVoIP)
		if err != nil || enc == nil {
			t.Fatalf("Error creating new encoder: %v", err)
		}
		addSine(pcm, SAMPLE_RATE, G4)
		data := make([]byte, 1000)
		n, err := enc.Encode(pcm, data)
		if err != nil {
			t.Fatalf("Couldn't encode data: %v", err)
		}
		return data[:n]
	}

	createDecoder := func(t *testing.T) *Decoder {
		dec, err := NewDecoder(SAMPLE_RATE, 1)
		if err != nil || dec == nil {
			t.Fatalf("Error creating new decoder: %v", err)
		}
		return dec
	}

	decodeInt16 := func(t *testing.T, data []byte, decodeSize int, expectOK bool) {
		dec := createDecoder(t)
		decodedMem := make([]int16, decodeSize+GUARD_SIZE*2)
		decodedRef := decodedMem[GUARD_SIZE : GUARD_SIZE+decodeSize : GUARD_SIZE+decodeSize]
		n, err := dec.Decode(data, decodedRef)
		checkGuardInt16(t, decodedMem[:GUARD_SIZE])
		checkGuardInt16(t, decodedMem[decodeSize+GUARD_SIZE:])
		checkResult(t, n, err, expectOK)
	}

	decodeFloat32 := func(t *testing.T, data []byte, decodeSize int, expectOK bool) {
		dec := createDecoder(t)
		decodedMem := make([]float32, decodeSize+GUARD_SIZE*2)
		decodedRef := decodedMem[GUARD_SIZE : GUARD_SIZE+decodeSize : GUARD_SIZE+decodeSize]
		n, err := dec.DecodeFloat32(data, decodedRef)
		checkGuardFloat32(t, decodedMem[:GUARD_SIZE])
		checkGuardFloat32(t, decodedMem[decodeSize+GUARD_SIZE:])
		checkResult(t, n, err, expectOK)
	}

	decodeFecInt16 := func(t *testing.T, data []byte, decodeSize int, expectOK bool) {
		dec := createDecoder(t)
		decodedMem := make([]int16, decodeSize+GUARD_SIZE*2)
		decodedRef := decodedMem[GUARD_SIZE : GUARD_SIZE+decodeSize : GUARD_SIZE+decodeSize]
		err := dec.DecodeFEC(data, decodedRef)
		checkGuardInt16(t, decodedMem[:GUARD_SIZE])
		checkGuardInt16(t, decodedMem[decodeSize+GUARD_SIZE:])
		checkResult(t, FRAME_SIZE, err, expectOK)
	}

	decodeFecFloat32 := func(t *testing.T, data []byte, decodeSize int, expectOK bool) {
		dec := createDecoder(t)
		decodedMem := make([]float32, decodeSize+GUARD_SIZE*2)
		decodedRef := decodedMem[GUARD_SIZE : GUARD_SIZE+decodeSize : GUARD_SIZE+decodeSize]
		err := dec.DecodeFECFloat32(data, decodedRef)
		checkGuardFloat32(t, decodedMem[:GUARD_SIZE])
		checkGuardFloat32(t, decodedMem[decodeSize+GUARD_SIZE:])
		checkResult(t, FRAME_SIZE, err, expectOK)
	}

	t.Run("smaller-buffer-int16", func(t *testing.T) {
		decodeInt16(t, encodeFrame(t), FRAME_SIZE-1, false)
	})

	t.Run("smaller-buffer-float32", func(t *testing.T) {
		decodeFloat32(t, encodeFrame(t), FRAME_SIZE-1, false)
	})

	t.Run("smaller-buffer-int16-fec", func(t *testing.T) {
		decodeFecFloat32(t, encodeFrame(t), FRAME_SIZE-1, false)
	})

	t.Run("smaller-buffer-float32-fec", func(t *testing.T) {
		decodeFecFloat32(t, encodeFrame(t), FRAME_SIZE-1, false)
	})

	t.Run("exact-buffer-int16", func(t *testing.T) {
		decodeInt16(t, encodeFrame(t), FRAME_SIZE, true)
	})

	t.Run("exact-buffer-float32", func(t *testing.T) {
		decodeFloat32(t, encodeFrame(t), FRAME_SIZE, true)
	})

	t.Run("exact-buffer-int16-fec", func(t *testing.T) {
		decodeFecInt16(t, encodeFrame(t), FRAME_SIZE, true)
	})

	t.Run("exact-buffer-float32-fec", func(t *testing.T) {
		decodeFecFloat32(t, encodeFrame(t), FRAME_SIZE, true)
	})

	t.Run("larger-buffer-int16", func(t *testing.T) {
		decodeInt16(t, encodeFrame(t), FRAME_SIZE+1, true)
	})

	t.Run("larger-buffer-float32", func(t *testing.T) {
		decodeFloat32(t, encodeFrame(t), FRAME_SIZE+1, true)
	})

	t.Run("larger-buffer-int16-fec", func(t *testing.T) {
		decodeFecInt16(t, encodeFrame(t), FRAME_SIZE+1, false)
	})

	t.Run("larger-buffer-float32-fec", func(t *testing.T) {
		decodeFecFloat32(t, encodeFrame(t), FRAME_SIZE+1, false)
	})
}
