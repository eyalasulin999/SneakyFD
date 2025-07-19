package marker

import (
	"golang.org/x/sys/unix"
)

type TCPOptionsMarker struct {
	MSS uint32
}

type TCPOptionsFlags struct {
	Timestamp bool
}

const (
	TIMESTAMP_BIT         = 1
	TIMESTAMP_HEADER_SIZE = 12
)

func parseTCPOptionsFlags(options uint8) (t TCPOptionsFlags) {
	t.Timestamp = options&TIMESTAMP_BIT != 0
	return
}

func calcMSSDiff(info *unix.TCPInfo) (diff uint32) {
	diff = 0

	options := parseTCPOptionsFlags(info.Options)
	if options.Timestamp {
		diff += TIMESTAMP_HEADER_SIZE
	}

	return
}

func (m TCPOptionsMarker) checkMSS(info *unix.TCPInfo) (marked bool) {
	diff := calcMSSDiff(info)
	marked = m.MSS == (info.Snd_mss + diff)
	return
}

func (m TCPOptionsMarker) Check(fd int) (marked bool) {
	marked = true

	info, err := unix.GetsockoptTCPInfo(fd, unix.IPPROTO_TCP, unix.TCP_INFO)
	if err != nil {
		return
	}

	if !m.checkMSS(info) {
		marked = false
		return
	}

	return
}
