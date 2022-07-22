package opus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	//ErrPacketNil       = errors.New("invalid nil segmentBuf")
	//ErrPayloadNil      = errors.New("invalid nil pageHeaderBuf")
	//ErrPayloadTooSmall = errors.New("header is not big enough")

	ErrStreamBeginNotFirst  = errors.New("stream begin not first")
	ErrBadPageHeader        = errors.New("bad page header")
	ErrNotOpusContent       = errors.New("not opus content")
	ErrMissingOpusTag       = errors.New("missing opus tag")
	ErrMalformedOpusTag     = errors.New("malformed opus tag")
	ErrInvalidOpusHead      = errors.New("invalid opus head")
	ErrOpusTagVendorTooLong = errors.New("opus tag vendor too long")
	ErrInvalidOpusTag       = errors.New("invalid opus tag")
	ErrTagPacketSpillover   = errors.New("tag packet spilled over")
	ErrPacketSizeLimit      = errors.New("packet size limit")
	ErrTooManyComments      = errors.New("too many comments")

	ErrChecksum = errors.New("pageHeaderBuf Checksum")
)

// FileReader reads from OGG OpusFile format.
type FileReader struct {
	stream io.Reader
	fd     io.ReadCloser

	eof bool

	// OpusHead
	head Head
	// OpusTag
	tag Tag

	// Current OggPage
	page PageHeader

	// Running CRC of page.
	// Does a Checksum check after the last segment is read.
	runningChecksum uint32

	// Ogg Page Header size.
	pageHeaderBuf [pageHeaderSize]byte

	segmentIndex  byte      // Current segment within page.
	segmentVector [255]byte // Payload buf
	segmentBuf    [255]byte // Payload buf
}

func (r *FileReader) Head() Head {
	return r.head
}

func (r *FileReader) Tag() Tag {
	return r.tag
}

func (r *FileReader) IsEOF() bool {
	return r.eof
}

func (r *FileReader) GranulePos() uint64 {
	return r.page.GranulePos
}

func (r *FileReader) Serial() uint32 {
	return r.page.Serial
}

func (r *FileReader) PageIndex() uint32 {
	return r.page.PageIndex
}

func (r *FileReader) ChannelCount() byte {
	return r.head.ChannelCount
}

func (r *FileReader) PreSkip() uint16 {
	return r.head.PreSkip
}

func (r *FileReader) SampleRate() uint32 {
	return r.head.SampleRate
}

func (r *FileReader) OutputGain() uint16 {
	return r.head.OutputGain
}

func (r *FileReader) ChannelMap() byte {
	return r.head.MappingFamily
}

// New builds a new OGG Opus writer
func OpenFile(fileName string) (*FileReader, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	writer, err := OpenReader(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	writer.fd = f
	return writer, nil
}

// OpenReader initialize a new OGG Opus writer with an io.Reader output
func OpenReader(out io.Reader) (*FileReader, error) {
	if out == nil {
		return nil, fmt.Errorf("file not opened")
	}

	writer := &FileReader{
		stream: out,
	}

	if closer, ok := out.(io.ReadCloser); ok {
		writer.fd = closer
	}

	if err := writer.readHeaders(); err != nil {
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
    |      ID header is contained on a single pageHeaderBuf
    |
    'Beginning Of Stream'

   Figure 1: Example Payload Organization for a Logical Ogg Opus Stream
*/

func (r *FileReader) readHeaders() error {
	err := r.gotoNextPage()
	if err != nil {
		return err
	}

	if r.page.HeaderType != pageHeaderTypeBeginningOfStream {
		return ErrStreamBeginNotFirst
	}

	// Read first packet.
	packet, err := r.readPacket()
	if len(packet) < 19 {
		return ErrInvalidOpusHead
	}
	if packet[0] != 'O' || packet[1] != 'p' || packet[2] != 'u' || packet[3] != 's' {
		return ErrInvalidOpusHead
	}
	if packet[4] != 'H' || packet[5] != 'e' || packet[6] != 'a' || packet[7] != 'd' {
		return ErrInvalidOpusHead
	}

	r.head.Version = packet[8]
	r.head.ChannelCount = packet[9]
	r.head.PreSkip = binary.LittleEndian.Uint16(packet[10:])
	r.head.SampleRate = binary.LittleEndian.Uint32(packet[12:])
	r.head.OutputGain = binary.LittleEndian.Uint16(packet[16:])
	r.head.MappingFamily = packet[18]

	// Go to the next page.
	err = r.gotoNextPage()
	if err != nil {
		return err
	}
	if r.page.GranulePos > 0 || r.page.HeaderType != pageHeaderTypeContinuationOfStream {
		return ErrMissingOpusTag
	}

	packet, err = r.readPacket()
	if err != nil {
		return err
	}

	if len(packet) < 16 {
		return ErrMalformedOpusTag
	}
	// Magic header "OpusTags"
	if packet[0] != 'O' || packet[1] != 'p' || packet[2] != 'u' || packet[3] != 's' ||
		packet[4] != 'T' || packet[5] != 'a' || packet[6] != 'g' || packet[7] != 's' {
		return ErrMissingOpusTag
	}

	vendorLength := binary.LittleEndian.Uint32(packet[8:])
	if vendorLength > 254 {
		return ErrOpusTagVendorTooLong
	}

	if vendorLength > 0 {
		if len(packet) < 16+int(vendorLength) {
			return ErrMalformedOpusTag
		}
		r.tag.Vendor = string(packet[12 : vendorLength+12])
	}

	index := int(12 + vendorLength)
	userCommentCount := binary.LittleEndian.Uint32(packet[index:])
	index += 4
	for userCommentCount > 0 {
		userCommentCount--
		if len(packet) < index+4 {
			return ErrMalformedOpusTag
		}
		commentLength := int(binary.LittleEndian.Uint32(packet[index:]))
		index += 4

		if len(packet) < commentLength+index {
			return ErrMalformedOpusTag
		}
		comment := string(packet[index : index+commentLength])
		index += commentLength
		r.tag.Comments = append(r.tag.Comments, comment)
	}

	// Go to first audio page.
	for {
		//isInPacket := r.segmentLenBuf[r.segmentIndex] == 255
		err = r.gotoNextPage()
		if err != nil {
			return err
		}

		if r.page.GranulePos > 0 {
			//if isInPacket {
			//	return ErrTagPacketSpillover
			//}
			break
		}
	}

	return nil
}

// Goes directly to the next page discarding all packets in the process.
func (r *FileReader) gotoNextPage() error {
	if r.segmentIndex < r.page.SegmentCount {
		if err := r.discardPage(); err != nil && err != io.EOF {
			return err
		}
	}
	header := r.pageHeaderBuf[:pageHeaderSize]
	n, err := r.stream.Read(header)

	if n < pageHeaderSize {
		if err != nil {
			return err
		}
		return io.ErrUnexpectedEOF
	}
	if err != nil {
		if err != io.EOF {
			return err
		}
		r.eof = true
	}

	if header[0] != 'O' || header[1] != 'g' || header[2] != 'g' || header[3] != 'S' {
		return ErrBadPageHeader
	}

	r.page.Version = header[4]
	r.page.HeaderType = header[5] // 1 = continuation, 2 = beginning of stream, 4 = end of stream
	r.page.GranulePos = binary.LittleEndian.Uint64(header[6:])
	r.page.Serial = binary.LittleEndian.Uint32(header[14:])
	r.page.PageIndex = binary.LittleEndian.Uint32(header[18:])
	r.page.Checksum = binary.LittleEndian.Uint32(header[22:])
	header[22] = 0
	header[23] = 0
	header[24] = 0
	header[25] = 0
	r.page.SegmentCount = header[26] // Set segment count.
	r.segmentIndex = 0               // Reset segment index.
	r.runningChecksum = 0

	// Calculate Checksum.
	for index := range header {
		r.runningChecksum = (r.runningChecksum << 8) ^ checksumTable[byte(r.runningChecksum>>24)^header[index]]
	}

	// Read Segment vector.
	n, err = r.stream.Read(r.segmentVector[:r.page.SegmentCount])
	if err != nil {
		return err
	}
	if n != int(r.page.SegmentCount) {
		return ErrBadPageHeader
	}
	// Compute checksum.
	for _, b := range r.segmentVector[:r.page.SegmentCount] {
		// Add segment length to Checksum.
		r.runningChecksum = (r.runningChecksum << 8) ^ checksumTable[byte(r.runningChecksum>>24)^b]
	}

	return nil
}

// Reads the next packet fully. For packets that are less than 255 bytes, no memory
// allocation is made, by utilizing the segment buffer internally. When using the
// accompanying oggwriter.FileWriter
//
// Each segment can be up to 255 bytes in length and is bounded by the page.
// A segment smaller than 255 bytes marks the end of a packet; the next segment
// will begin a new packet. A packet with a length that is a multiple of 255 bytes
// will end with a segment of length 0. If the last segment of the page is 255 bytes
// then the last packet is continued on the following page.
func (r *FileReader) readPacket() ([]byte, error) {
	var packet []byte
	for {
		// Read next segment.
		segment, err := r.readSegment()

		// End of multi-segment or zero sized packet?
		if len(segment) == 0 {
			if err != nil {
				if err == io.EOF {
					// See if this is the last packet.
					err = r.gotoNextPage()
					if err != nil {
						// Last packet.
						return packet, err
					}
					continue
					// Not the last packet, just an empty packet.
					// Return it an remove the error.
					//return packet, nil
				} else {
					// Fatal error.
					return packet, err
				}
			}
			// End of packet.
			return packet, nil
		}

		// Is this the end of a packet?
		if len(segment) < 255 {
			// Payload < 255
			if packet == nil {
				// Segment is entire packet.
				return segment, err
			} else {
				// Need to copy and append to packet.
				// Segment buffers are re-used.
				packet = append(packet, segment...)
				if len(packet) > packetMaxSize {
					return packet, ErrPacketSizeLimit
				}

				if err != nil {
					if err == io.EOF {
						// Ignore page EOF.
						return packet, nil
					} else {
						// Report fatal error.
						return packet, err
					}
				}
				return packet, nil
			}
		} else {
			// Need to copy and append to packet.
			// Segment buffers are re-used.
			packet = append(packet, segment...)
			if len(packet) > packetMaxSize {
				return packet, ErrPacketSizeLimit
			}

			if err != nil {
				if err == io.EOF {
					err = r.gotoNextPage()
					if err != nil {
						return packet, err
					}
					continue
				} else {
					return packet, err
				}
			}
		}
	}
}

// Goes to the start of the next page discarding all bytes along the way.
func (r *FileReader) discardPage() error {
	for {
		_, err := r.readSegment()
		if err != nil {
			return err
		}
	}
}

// Reads the next segment in the page. Returns EOF when current page is finished.
// If the underlying stream returns EOF this is treated as an io.ErrUnexpectedEOF.
func (r *FileReader) readSegment() ([]byte, error) {
	if r.segmentIndex >= r.page.SegmentCount {
		return nil, io.EOF
	}

	segmentLength := r.segmentVector[r.segmentIndex]
	r.segmentIndex++

	//// Increase segment index.
	//r.segmentIndex++
	//
	//// Read next segment length.
	//n, err := r.stream.Read(r.segmentLenBuf[:])
	//if n <= 0 {
	//	return nil, err
	//}
	//
	//// Get length.
	//segmentLength := r.segmentVector[r.se]
	//
	//if err != nil {
	//	if err == io.EOF {
	//		if segmentLength > 0 {
	//			return nil, io.ErrUnexpectedEOF
	//		} else {
	//			return nil, io.EOF
	//		}
	//	}
	//	return nil, err
	//}

	// Add segment length to Checksum.
	//r.runningChecksum = (r.runningChecksum << 8) ^ checksumTable[byte(r.runningChecksum>>24)^segmentLength]
	if segmentLength == 0 {
		return nil, nil
	}

	buf := r.segmentBuf[0:segmentLength]

	// Read segment from stream.
	n, err := r.stream.Read(buf)

	// Unexpected EOF?
	if len(buf) != n {
		if err != nil {
			return buf[0:n], err
		}
		return buf[0:n], io.ErrUnexpectedEOF
	}

	// Compute page Checksum.
	for index := range buf {
		r.runningChecksum = (r.runningChecksum << 8) ^ checksumTable[byte(r.runningChecksum>>24)^buf[index]]
	}

	// Should we validate Checksum?
	if r.segmentIndex == r.page.SegmentCount {
		// Checksum check
		if r.runningChecksum != r.page.Checksum {
			return buf, ErrChecksum
		}
	}

	return buf, err
}

// Read the next packet.
func (r *FileReader) ReadPacket() ([]byte, error) {
	if r.stream == nil {
		return nil, os.ErrClosed
	}
	return r.readPacket()
}

// Close stops the recording
func (r *FileReader) Close() error {
	// Returns no error has it may be convenient to call
	// Close() multiple times
	if r.fd == nil {
		return nil
	}
	err := r.fd.Close()
	r.fd = nil
	return err
}
