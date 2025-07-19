package monitor

import (
	"time"

	"sneakyfd/internal/procnet"
	"sneakyfd/internal/types"
)

type Monitor struct {
	DstPorts      types.Ports
	SrcPorts      types.Ports
	CheckInterval time.Duration
	prevSocks     map[types.SocketInode]types.Socket
}

func (m *Monitor) getEstablishedSockets() (socks types.Sockets, err error) {
	socks, err = procnet.GetTCPSockets(func(sock *types.Socket) bool {
		return sock.State == types.StateEstablished && m.DstPorts.Contains(sock.LocalAddr.Port) && m.SrcPorts.Contains(sock.RemoteAddr.Port)
	})
	return
}

func (m *Monitor) GetNewEstablishedSockets() (newSocks types.Sockets, err error) {
	currSocks, err := m.getEstablishedSockets()
	if err != nil {
		return
	}

	newSocks = types.Sockets{}
	currMap := make(map[types.SocketInode]types.Socket)

	for _, sock := range currSocks {
		currMap[sock.Inode] = sock
		if _, exists := m.prevSocks[sock.Inode]; !exists {
			newSocks = append(newSocks, sock)
		}
	}

	m.prevSocks = currMap
	return
}

func (m *Monitor) Init() (err error) {
	_, err = m.GetNewEstablishedSockets()
	return
}
