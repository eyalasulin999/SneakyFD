package procnet

import (
	"os"

	"sneakyfd/internal/types"
)

const (
	pathProcTCP = "/proc/net/tcp"
	ipv4StrLen  = 8
)

type Filter func(*types.Socket) bool

func NoFilter(*types.Socket) bool { return true }

func GetTCPSockets(filter Filter) (socks types.Sockets, err error) {
	f, err := os.Open(pathProcTCP)
	if err != nil {
		return nil, err
	}
	tabs, err := parseSocktab(f, filter)
	f.Close()
	if err != nil {
		return nil, err
	}
	return tabs, nil
}
