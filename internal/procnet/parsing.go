package procnet

import (
	"net"
	"strconv"
	"encoding/binary"
	"io"
	"bufio"
	"strings"
	"fmt"

	"sneakyfd/internal/types"
)

func parseIPv4(s string) (IP net.IP, err error) {
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, err
	}
	ip := make(net.IP, net.IPv4len)
	binary.LittleEndian.PutUint32(ip, uint32(v))
	return ip, nil
}

func parseAddr(s string) (*types.SocketAddr, error) {
	fields := strings.Split(s, ":")
	if len(fields) < 2 {
		return nil, fmt.Errorf("netstat: not enough fields: %v", s)
	}
	var ip net.IP
	var err error
	switch len(fields[0]) {
	case ipv4StrLen:
		ip, err = parseIPv4(fields[0])
	default:
		err = fmt.Errorf("netstat: bad formatted string: %v", fields[0])
	}
	if err != nil {
		return nil, err
	}
	v, err := strconv.ParseUint(fields[1], 16, 16)
	if err != nil {
		return nil, err
	}
	return &types.SocketAddr{IP: ip, Port: uint16(v)}, nil
}


func parseSocktab(r io.Reader, accept Filter) (types.Sockets, error) {
	br := bufio.NewScanner(r)
	tab := make(types.Sockets, 0, 4)

	// Discard title
	br.Scan()

	for br.Scan() {
		var e types.Socket
		line := br.Text()
		// Skip comments
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) < 12 {
			return nil, fmt.Errorf("netstat: not enough fields: %v, %v", len(fields), fields)
		}
		addr, err := parseAddr(fields[1])
		if err != nil {
			return nil, err
		}
		e.LocalAddr = addr
		addr, err = parseAddr(fields[2])
		if err != nil {
			return nil, err
		}
		e.RemoteAddr = addr
		u, err := strconv.ParseUint(fields[3], 16, 8)
		if err != nil {
			return nil, err
		}
		e.State = types.SocketState(u)
		u, err = strconv.ParseUint(fields[7], 10, 32)
		if err != nil {
			return nil, err
		}
		e.UID = uint32(u)
		e.Inode =  types.SocketInode(fields[9])
		if accept(&e) {
			tab = append(tab, e)
		}
	}
	return tab, br.Err()
}