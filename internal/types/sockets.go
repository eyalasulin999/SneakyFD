package types

import (
	"fmt"
	"net"

	"github.com/rs/zerolog"
)

type SocketInode string

type SocketAddr struct {
	IP   net.IP
	Port uint16
}

func (s *SocketAddr) String() string {
	return fmt.Sprintf("%v:%d", s.IP, s.Port)
}

type SocketState uint8

const (
	StateEstablished SocketState = 0x01
	StateSynSent     SocketState = 0x02
	StateSynRecv     SocketState = 0x03
	StateFinWait1    SocketState = 0x04
	StateFinWait2    SocketState = 0x05
	StateTimeWait    SocketState = 0x06
	StateClose       SocketState = 0x07
	StateCloseWait   SocketState = 0x08
	StateLastAck     SocketState = 0x09
	StateListen      SocketState = 0x0a
	StateClosing     SocketState = 0x0b
)

type Socket struct {
	Inode      SocketInode
	LocalAddr  *SocketAddr
	RemoteAddr *SocketAddr
	State      SocketState
	UID        uint32
}

type Sockets []Socket

type SocketProc struct {
	PID int
	FD  int
}

func (s Socket) MarshalZerologObject(e *zerolog.Event) {
	e.Str("inode", string(s.Inode)).
		Str("src_ip", s.RemoteAddr.IP.String()).
		Uint16("src_port", s.RemoteAddr.Port).
		Str("dst_ip", s.LocalAddr.IP.String()).
		Uint16("dst_port", s.LocalAddr.Port)
}

func (s SocketProc) MarshalZerologObject(e *zerolog.Event) {
	e.Int("pid", s.PID).Int("fd", s.FD)
}
