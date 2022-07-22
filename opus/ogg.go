package opus

const (
	pageHeaderTypeContinuationOfStream = 0x00
	pageHeaderTypeBeginningOfStream    = 0x02
	pageHeaderTypeEndOfStream          = 0x04
	defaultPreSkip                     = 3840 // 3840 recommended in the RFC
	idPageSignature                    = "OpusHead"
	commentPageSignature               = "OpusTags"
	pageHeaderSignature                = "OggS"
	pageHeaderSize                     = 27

	pageSize      = 8192 - pageHeaderSize - (8192/255 - 1) // 4k max pages including the header.
	packetMaxSize = 62000
)

// OGG Head page payload.
type Head struct {
	Version       byte
	ChannelCount  byte
	PreSkip       uint16
	SampleRate    uint32
	MappingFamily byte
	OutputGain    uint16
}

//
type Tag struct {
	Vendor   string
	Comments []string
}

type ServerInfo struct {
	Name        string
	Description string
	Genre       string
	URL         string
	Server      string
	ContentType string
	BitrateKbps int32
	IsPublic    bool
	IsSSL       bool
}

type Page struct {
	PageHeader
	Content []byte
}

type PageHeader struct {
	Version      byte
	HeaderType   byte
	GranulePos   uint64
	Serial       uint32
	PageIndex    uint32
	Checksum     uint32
	SegmentCount byte
}

var checksumTable = generateChecksumTable()

func generateChecksumTable() [256]uint32 {
	var table [256]uint32
	const poly = 0x04c11db7

	for i := range table {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if (r & 0x80000000) != 0 {
				r = (r << 1) ^ poly
			} else {
				r <<= 1
			}
			table[i] = r & 0xffffffff
		}
	}
	return table
}
