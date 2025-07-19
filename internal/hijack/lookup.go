package hijack

import (
	"os"
	"path/filepath"
	"strconv"
	"errors"

	"sneakyfd/internal/types"
)

func LookupSocket(inode types.SocketInode) (socketProc types.SocketProc, err error) {
	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}

	target := "socket:[" + inode + "]"

	for _, entry := range procEntries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // not a PID directory
		}

		fdDir := filepath.Join("/proc", entry.Name(), "fd")
		fdEntries, err := os.ReadDir(fdDir)
		if err != nil {
			continue // process might have exited or permission denied
		}

		for _, fdEntry := range fdEntries {
			fdPath := filepath.Join(fdDir, fdEntry.Name())
			linkTarget, err := os.Readlink(fdPath)
			if err != nil {
				continue // can't read symlink
			}

			if linkTarget == string(target) {
				fd, err := strconv.Atoi(fdEntry.Name())
				if err != nil {
					continue
				}
				return types.SocketProc{PID: pid, FD: fd}, nil
			}
		}
	}

	err = errors.New("socket inode not found in /proc")
	return
}
