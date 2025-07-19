package process

import (
	"golang.org/x/sys/unix"
)

func KillProcess(pid int) (killed bool) {
	killed = false

	err := unix.Kill(pid, unix.SIGKILL)
	if err != nil {
		return
	}

	killed = true
	return
}
