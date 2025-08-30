package process

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

const (
	INTERVAL = 1 * time.Second
)

func WaitProcess(pid int, fd int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			// check if process terminated
			err := unix.Kill(pid, 0)
			if err == unix.ESRCH {
				return true
			} else if err != nil {
				return false
			}

			// check if socket closed
			fdPath := fmt.Sprintf("/proc/%d/fd/%d", pid, fd)
			_, err = os.Stat(fdPath)
			if os.IsNotExist(err) {
				return true
			} else if err != nil {
				return false
			}
		}
	}
}
