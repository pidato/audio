package opus

//
//import (
//	"bytes"
//	"fmt"
//	"github.com/pidato/audio/pcm"
//	"github.com/pion/rtp/v2"
//	"github.com/pion/rtp/v2/codecs"
//	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
//	"io"
//	"math/rand"
//	"os"
//	"testing"
//
//	//"github.com/pion/rtp"
//	"github.com/stretchr/testify/assert"
//)
//
//type oggWriterPacketTest struct {
//	buffer       io.FileWriter
//	message      string
//	messageClose string
//	packet       *rtp.Payload
//	writer       *FileWriter
//	err          error
//	closeErr     error
//}
//
//func TestOpenWriter(t *testing.T) {
//	file, err := os.Open("testdata/recording.wav")
//	if err != nil {
//		t.Fatal(err)
//	}
//	ptime := 20
//	reader, err := pcm.OpenWav(file, ptime)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	sampleRate := reader.SampleRate()
//	sampleRateAdjustment := float32(48000) / float32(sampleRate)
//	opusFrameRate := int(float32(reader.FrameSize()) * sampleRateAdjustment)
//	_ = opusFrameRate
//
//	frameSize := reader.FrameSize()
//	_ = frameSize
//	//frameSize := int((int64(ptime) * int64(time.Millisecond)) / int64(sampleDuration))
//
//	enc, err := NewEncoder(sampleRate, 1, AppVoIP)
//	if err != nil || enc == nil {
//		t.Fatalf("Error creating new encoder: %v", err)
//	}
//
//	//opusSampleRate, err := enc.SampleRate()
//	//if err != nil {
//	//	t.Fatal(err)
//	//}
//	//fmt.Println(opusSampleRate)
//	//
//	//err = enc.SetComplexity(10)
//	//if err != nil {
//	//	t.Fatal(err)
//	//}
//	err = enc.SetMaxBandwidth(Wideband)
//	if err != nil {
//		t.Fatal(err)
//	}
//	//err = enc.SetBitrate(16000)
//	//if err != nil {
//	//	t.Fatal(err)
//	//}
//	err = enc.SetDTX(true)
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = enc.SetInBandFEC(true)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	packetizer := rtp.NewPacketizer(
//		1200,
//		111,
//		rand.Uint32(),
//		&codecs.OpusPayloader{},
//		rtp.NewFixedSequencer(0),
//		48000,
//	)
//	_ = packetizer
//
//	ow, err := oggwriter.New("testdata/2.opus", 48000, 1)
//	if err != nil {
//		t.Fatal(err)
//	}
//	opusWriter, err := CreateFile("testdata/1.opus", uint32(sampleRate), 1, 0)
//	if err != nil {
//		t.Fatal(err)
//	}
//	//opusFrames := make([][]byte, 0, 128)
//
//	eof := false
//	for !eof {
//		frame, err := reader.ReadFrame()
//		if err != nil {
//			eof = err == io.EOF
//			if !eof {
//				break
//			}
//		}
//		if len(frame) == 0 {
//			err = opusWriter.WriteEOF(nil, 0)
//			if err != nil {
//				t.Fatal(err)
//			}
//			break
//		}
//		if len(frame) != frameSize {
//			l := len(frame)
//			frame = frame[0:frameSize]
//			for i := l; i < len(frame); i++ {
//				frame[i] = 0
//			}
//		}
//		opus := make([]byte, frameSize*2)
//		n, err := enc.Encode(frame, opus)
//		if err != nil {
//			t.Fatalf("Error creating new encoder: %v", err)
//		}
//
//		opusFrame := opus[:n]
//		//opusFrames = append(opusFrames, opusFrame)
//
//		//packets := packetizer.Packetize(opusFrame, uint32(len(frame)))
//		packets := packetizer.Packetize(opusFrame, uint32(opusFrameRate))
//		for _, packet := range packets {
//			err = ow.WriteRTP(packet)
//			if err != nil {
//				t.Fatal(err)
//			}
//		}
//
//		if eof {
//			err = opusWriter.WriteEOF(opusFrame, opusFrameRate)
//			//err = opusWriter.WriteEOF(opusFrame, len(frame))
//		} else {
//			err = opusWriter.Write(opusFrame, opusFrameRate)
//			//err = opusWriter.Write(opusFrame, len(frame))
//		}
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	err = ow.Close()
//	if err != nil {
//		t.Fatal(err)
//	}
//	err = opusWriter.Close()
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//func TestOggWriter_AddPacketAndClose(t *testing.T) {
//	rawPkt := []byte{
//		0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
//		0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
//	}
//
//	validPacket := &rtp.Payload{
//		Header: rtp.Header{
//			Marker:           true,
//			Extension:        true,
//			ExtensionProfile: 1,
//			ExtensionPayload: []byte{0xFF, 0xFF, 0xFF, 0xFF},
//			Version:          2,
//			PayloadOffset:    20,
//			PayloadType:      111,
//			SequenceNumber:   27023,
//			Timestamp:        3653407706,
//			SSRC:             476325762,
//			CSRC:             []uint32{},
//		},
//		Payload: rawPkt[20:],
//		Raw:     rawPkt,
//	}
//
//	assert := assert.New(t)
//
//	// The linter misbehave and thinks this code is the same as the tests in ivf-writer_test
//	// nolint:dupl
//	addPacketTestCase := []oggWriterPacketTest{
//		{
//			buffer:       &bytes.Buffer{},
//			message:      "FileWriter shouldn't be able to write something to a closed file",
//			messageClose: "FileWriter should be able to close an already closed file",
//			packet:       validPacket,
//			err:          fmt.Errorf("file not opened"),
//			closeErr:     nil,
//		},
//		{
//			buffer:       &bytes.Buffer{},
//			message:      "FileWriter shouldn't be able to write an empty packet",
//			messageClose: "FileWriter should be able to close the file",
//			packet:       &rtp.Payload{},
//			err:          fmt.Errorf("invalid nil packet"),
//			closeErr:     nil,
//		},
//		{
//			buffer:       &bytes.Buffer{},
//			message:      "FileWriter should be able to write an Opus packet",
//			messageClose: "FileWriter should be able to close the file",
//			packet:       validPacket,
//			err:          nil,
//			closeErr:     nil,
//		},
//		{
//			buffer:       nil,
//			message:      "FileWriter shouldn't be able to write something to a closed file",
//			messageClose: "FileWriter should be able to close an already closed file",
//			packet:       nil,
//			err:          fmt.Errorf("file not opened"),
//			closeErr:     nil,
//		},
//	}
//
//	// First test case has a 'nil' file descriptor
//	writer, err := OpenWriter(addPacketTestCase[0].buffer, 48000, 2, 0, Tag{})
//	assert.Nil(err, "FileWriter should be created")
//	assert.NotNil(writer, "FileWriter shouldn't be nil")
//	err = writer.Close()
//	assert.Nil(err, "FileWriter should be able to close the file descriptor")
//	writer.stream = nil
//	addPacketTestCase[0].writer = writer
//
//	// Second test writes tries to write an empty packet
//	writer, err = OpenWriter(addPacketTestCase[1].buffer, 48000, 2, 0, Tag{})
//	assert.Nil(err, "FileWriter should be created")
//	assert.NotNil(writer, "FileWriter shouldn't be nil")
//	addPacketTestCase[1].writer = writer
//
//	// Third test writes tries to write a valid Opus packet
//	writer, err = OpenWriter(addPacketTestCase[2].buffer, 48000, 2, 0, Tag{})
//	assert.Nil(err, "FileWriter should be created")
//	assert.NotNil(writer, "FileWriter shouldn't be nil")
//	addPacketTestCase[2].writer = writer
//
//	// Fourth test tries to write to a nil stream
//	writer, err = OpenWriter(addPacketTestCase[3].buffer, 4800, 2, 0, Tag{})
//	assert.NotNil(err, "IVFWriter shouldn't be created")
//	assert.Nil(writer, "FileWriter should be nil")
//	addPacketTestCase[3].writer = writer
//
//	//for _, t := range addPacketTestCase {
//	//	if t.writer != nil {
//	//		res := t.writer.WriteRTP(t.packet)
//	//		assert.Equal(t.err, res, t.message)
//	//	}
//	//}
//
//	for _, t := range addPacketTestCase {
//		if t.writer != nil {
//			res := t.writer.Close()
//			assert.Equal(t.closeErr, res, t.messageClose)
//		}
//	}
//}
