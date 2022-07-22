package opus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
)

var (
	ErrPacketNil       = errors.New("invalid nil packet")
	ErrPayloadNil      = errors.New("invalid nil payload")
	ErrPayloadTooSmall = errors.New("payload is not big enough")
	ErrFrameSize       = errors.New("frame size")

	defaultVendor = "?????"
)

// FileWriter is used to take Opus RTP packets or raw Opus packets and write them to an OGG on disk
type FileWriter struct {
	stream io.Writer
	fd     io.WriteCloser

	head          Head
	tag           Tag
	serial        uint32
	pageIndex     uint32
	eof           bool
	pageHeaderBuf [pageHeaderSize + 1]byte
	page          PageHeader
	writerIndex   int
	maxSegments   byte
	sampleCount   uint64

	// Page buffer.
	pageBuffer    [pageSize]byte
	segmentVector [255]byte

	mu sync.Mutex
}

func CreateFile(fileName string, sampleRate uint32, channelCount uint16, preSkip uint16) (*FileWriter, error) {
	return CreateFileWithTag(fileName, sampleRate, channelCount, preSkip, Tag{Vendor: defaultVendor})
}

// CreateFile builds a new OGG Opus writer
func CreateFileWithTag(fileName string, sampleRate uint32, channelCount uint16, preSkip uint16, tag Tag) (*FileWriter, error) {
	f, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	writer, err := OpenWriter(f, sampleRate, channelCount, preSkip, tag)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	writer.fd = f
	return writer, nil
}

// OpenWriter initialize a new OGG Opus writer with an io.Writer output
func OpenWriter(out io.Writer, sampleRate uint32, channelCount uint16, preSkip uint16, tag Tag) (*FileWriter, error) {
	if out == nil {
		return nil, fmt.Errorf("file not opened")
	}

	writer := &FileWriter{
		stream:      out,
		serial:      rand.Uint32(),
		maxSegments: 100,
		tag:         tag,
	}

	writer.head.SampleRate = sampleRate
	writer.head.ChannelCount = byte(channelCount)
	writer.head.PreSkip = preSkip

	// page headers starts with 'OggS'
	writer.pageHeaderBuf[0] = 'O'
	writer.pageHeaderBuf[1] = 'g'
	writer.pageHeaderBuf[2] = 'g'
	writer.pageHeaderBuf[3] = 'S'
	//copy(writer.pageHeaderBuf[0:], pageHeaderSignature)                     // page headers starts with 'OggS'

	writer.pageHeaderBuf[4] = 0 // Version

	binary.LittleEndian.PutUint32(writer.pageHeaderBuf[14:], writer.serial) // Bitstream Serial number
	//writer.pageHeaderBuf[26] = 1                                            // Number of segments in page, giving always 1 segment

	if err := writer.writeHeaders(); err != nil {
		return nil, err
	}

	return writer, nil
}

/*
    ref: https://tools.ietf.org/html/rfc7845.html
    https://git.xiph.org/?p=opus-tools.git;a=blob;f=src/opus_header.c#l219

       Page 0         Pages 1 ... n        Pages (n+1) ...
    +------------+ +---+ +---+ ... +---+ +-----------+ +---------+ +--
    |            | |   | |   |     |   | |           | |         | |
    |+----------+| |+-----------------+| |+-------------------+ +-----
    |||ID Header|| ||  Comment Header || ||Audio Data Payload 1| | ...
    |+----------+| |+-----------------+| |+-------------------+ +-----
    |            | |   | |   |     |   | |           | |         | |
    +------------+ +---+ +---+ ... +---+ +-----------+ +---------+ +--
    ^      ^                           ^
    |      |                           |
    |      |                           Mandatory Page Break
    |      |
    |      ID header is contained on a single page
    |
    'Beginning Of Stream'

   Figure 1: Example Payload Organization for a Logical Ogg Opus Stream
*/

func (w *FileWriter) writeHeaders() error {
	w.sampleCount = 0

	// Stack allocate.
	var header [24]byte

	{
		// ID Header
		oggIDHeader := header[0:19]

		copy(oggIDHeader[0:], idPageSignature)      // Magic Signature 'OpusHead'
		oggIDHeader[8] = 1                          // Version
		oggIDHeader[9] = uint8(w.head.ChannelCount) // Channel count
		//binary.LittleEndian.PutUint16(oggIDHeader[10:], defaultPreSkip/4)    // pre-skip
		binary.LittleEndian.PutUint16(oggIDHeader[10:], w.head.PreSkip)    // pre-skip
		binary.LittleEndian.PutUint32(oggIDHeader[12:], w.head.SampleRate) // original sample rate, any valid sample e.g 48000
		binary.LittleEndian.PutUint16(oggIDHeader[16:], 0)                 // output gain
		oggIDHeader[18] = 0                                                // channel map 0 = one stream: mono or stereo

		w.page.SegmentCount = 1
		w.segmentVector[0] = byte(len(oggIDHeader))
		// Reference: https://tools.ietf.org/html/rfc7845.html#page-6
		// RFC specifies that the ID Header page should have a granule position of 0 and a Header kind set to 2 (StartOfStream)
		err := w.writePage(oggIDHeader, pageHeaderTypeBeginningOfStream, 0)
		if err != nil {
			return err
		}
		w.pageIndex++
	}

	//size := len(commentPageSignature) + len(w.tag.Vendor) + 8 + (len(w.tag.Comments) * 4)
	//for _, v := range w.tag.Comments {
	//	size += len(v)
	//}
	//
	//b := pbytes.GetLen(size)
	//buf := bytes.NewBuffer(b)
	//
	//buf.WriteString(commentPageSignature)
	//binary.LittleEndian.PutUint32(header[0:], uint32(len(w.tag.Vendor)))
	//buf.Write(header[0:4])
	//buf.WriteString(w.tag.Vendor)
	//binary.LittleEndian.PutUint32(header[0:], uint32(len(w.tag.Comments)))
	//buf.Write(header[0:4])
	//
	//// Return bytes to pool.
	//pbytes.Put(b)

	{
		// Comment Header
		oggCommentHeader := header[0:21]

		copy(oggCommentHeader[0:], commentPageSignature)        // Magic Signature 'OpusTags'
		binary.LittleEndian.PutUint32(oggCommentHeader[8:], 5)  // Vendor Length
		copy(oggCommentHeader[12:], "?????")                    // Vendor name '?????'
		binary.LittleEndian.PutUint32(oggCommentHeader[17:], 0) // User Comment List Length

		w.page.SegmentCount = 1
		w.segmentVector[0] = byte(len(oggCommentHeader))

		// RFC specifies that the page where the CommentHeader completes should have a granule position of 0
		err := w.writePage(oggCommentHeader, pageHeaderTypeContinuationOfStream, 0)
		if err != nil {
			return err
		}
		w.pageIndex++
	}

	w.page.Version = 0
	w.page.HeaderType = pageHeaderTypeContinuationOfStream
	w.page.Checksum = 0
	w.page.SegmentCount = 0
	w.page.GranulePos = 0

	return nil
}

func (w *FileWriter) writePage(payload []byte, headerType uint8, granulePos uint64) error {
	if w.stream == nil {
		return os.ErrClosed
	}

	if w.page.SegmentCount == 0 {
		w.page.SegmentCount = 1
		if len(payload) > 254 {
			return io.ErrShortBuffer
		}
		w.segmentVector[0] = byte(len(payload))
	}

	pageHeader := w.pageHeaderBuf[:27]

	//copy(pageHeader[0:], pageHeaderSignature)                 // page headers starts with 'OggS'
	//pageHeader[4] = 0                                         // Version
	pageHeader[5] = headerType                                // 1 = continuation, 2 = beginning of stream, 4 = end of stream
	binary.LittleEndian.PutUint64(pageHeader[6:], granulePos) // granule position
	//binary.LittleEndian.PutUint32(pageHeader[14:], w.serial)  // Bitstream Serial number
	binary.LittleEndian.PutUint32(pageHeader[18:], w.pageIndex) // Page sequence number
	pageHeader[22] = 0
	pageHeader[23] = 0
	pageHeader[24] = 0
	pageHeader[25] = 0
	pageHeader[26] = w.page.SegmentCount // Number of segments in page, giving always 1 segment

	// Calculate Checksum.
	var checksum uint32
	for index := range pageHeader {
		checksum = (checksum << 8) ^ checksumTable[byte(checksum>>24)^pageHeader[index]]
	}
	if w.page.SegmentCount > 0 {
		for index := 0; index < int(w.page.SegmentCount); index++ {
			checksum = (checksum << 8) ^ checksumTable[byte(checksum>>24)^w.segmentVector[index]]
		}
	}
	if len(payload) > 0 {
		for index := range payload {
			checksum = (checksum << 8) ^ checksumTable[byte(checksum>>24)^payload[index]]
		}
	}
	binary.LittleEndian.PutUint32(pageHeader[22:], checksum) // Checksum - generating for page data and inserting at 22th position into 32 bits

	// Write PageHeader.
	n, err := w.stream.Write(pageHeader)
	if err != nil {
		return err
	}
	if n != len(pageHeader) {
		return io.ErrShortWrite
	}

	if w.page.SegmentCount > 0 {
		n, err = w.stream.Write(w.segmentVector[:w.page.SegmentCount])
		if err != nil {
			return err
		}
		if n != int(w.page.SegmentCount) {
			return io.ErrShortWrite
		}
	}

	if len(payload) > 0 {
		// Write Payload.
		n, err = w.stream.Write(payload)
		if err != nil {
			return err
		}
		if n != len(payload) {
			return io.ErrShortWrite
		}
	}
	return nil
}

func (w *FileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushPage()
}

func (w *FileWriter) flushPage() error {
	if w.writerIndex == 0 {
		return nil
	}

	if err := w.writePage(
		w.pageBuffer[:w.writerIndex],
		w.page.HeaderType,
		//w.page.GranulePos,
		w.sampleCount,
	); err != nil {
		return err
	}

	w.pageIndex++
	w.page.PageIndex = w.pageIndex
	w.page.GranulePos = w.sampleCount
	w.page.SegmentCount = 0
	w.page.HeaderType = pageHeaderTypeContinuationOfStream
	w.page.Checksum = 0
	w.writerIndex = 0
	return nil
}

func (w *FileWriter) flushEOF() error {
	return w.writePage(
		w.pageBuffer[:w.writerIndex],
		pageHeaderTypeEndOfStream,
		w.sampleCount,
	)
}

func (w *FileWriter) writeSegments(packet []byte, samples int) error {
	if len(packet) == 0 {
		return nil
	}

	// Ensure frame size is valid.
	switch samples {
	case 120: // 2.5ms
	case 240: // 5ms
	case 480: // 10ms
	case 960: // 20ms
	case 1920: // 40ms
	case 2880: // 60ms
	case 2880 * 2: // 120ms
	default:
		return ErrFrameSize
	}
	// Is it multiple segments?
	if len(packet) >= 255 {
		segments := len(packet) / 255
		extra := len(packet) % 255

		newSegmentCount := segments + int(w.page.SegmentCount)
		if extra > 0 {
			newSegmentCount++
		}

		if newSegmentCount > int(w.maxSegments) || (len(packet) > len(w.pageBuffer)-w.writerIndex-segments-1) {
			err := w.flushPage()
			if err != nil {
				return err
			}
		}

		if w.eof {
			w.page.HeaderType = pageHeaderTypeEndOfStream
		}

		// Don't accept packets larger than what a single page buffer can hold.
		if len(packet) > len(w.pageBuffer) {
			return ErrPacketSizeLimit
		}

		w.page.SegmentCount += byte(segments)
		for i := 0; i < segments; i++ {
			w.segmentVector[w.page.SegmentCount] = 255
			w.page.SegmentCount++
			p := packet[i*255 : (i*255)+255]
			copy(w.pageBuffer[w.writerIndex:], p)
			w.writerIndex += 255
		}

		if extra > 0 {
			w.segmentVector[w.page.SegmentCount] = byte(extra)
			w.page.SegmentCount++
			p := packet[255*segments:]
			copy(w.pageBuffer[w.writerIndex:], p)
			w.writerIndex += len(p)
		}

		// Increment sample count.
		w.sampleCount += uint64(samples)

		// Flush at least once per second.
		if w.sampleCount%xMAX_BITRATE == 0 || w.eof {
			if err := w.flushPage(); err != nil {
				return err
			}
		}
	} else {
		if w.page.SegmentCount == w.maxSegments || len(packet) > len(w.pageBuffer)-w.writerIndex-2 {
			err := w.flushPage()
			if err != nil {
				return err
			}
		}

		if w.eof {
			w.page.HeaderType = pageHeaderTypeEndOfStream
		}

		// Increment sample count.
		w.sampleCount += uint64(samples)

		// Increase segments.
		w.segmentVector[w.page.SegmentCount] = byte(len(packet))
		w.page.SegmentCount++
		copy(w.pageBuffer[w.writerIndex:w.writerIndex+len(packet)], packet)
		w.writerIndex += len(packet)

		// Flush at least once per second.
		if w.sampleCount%xMAX_BITRATE == 0 || w.eof {
			if err := w.flushPage(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *FileWriter) Write(packet []byte, samples int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(packet) == 0 {
		return ErrFrameSize
	}
	if err := w.writeSegments(packet, samples); err != nil {
		return err
	}
	return nil
}

func (w *FileWriter) WriteEOF(packet []byte, samples int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.eof {
		return os.ErrClosed
	}
	w.eof = true
	if err := w.writeSegments(packet, samples); err != nil {
		return err
	}
	return nil
}

// Close stops the recording
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Returns no error has it may be convenient to call
	// Close() multiple times
	if w.fd == nil {
		return os.ErrClosed
	}

	if w.eof {
		if w.fd == nil {
			return os.ErrClosed
		}
		defer func() {
			w.fd = nil
		}()
		return w.fd.Close()
	}

	// Write empty EOF page.
	return w.WriteEOF(nil, 0)
}
