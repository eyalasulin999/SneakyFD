package hijack

import (
	"golang.org/x/sys/unix"

	"sneakyfd/internal/types"
)

func HijackSocket(socketProc types.SocketProc) (fd int, err error) {
	pidFd, err := unix.PidfdOpen(socketProc.PID, 0)
	if err != nil {
		return
	}
	defer unix.Close(pidFd)

	fd, err = unix.PidfdGetfd(pidFd, socketProc.FD, 0)
	return
}
