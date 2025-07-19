package process

import (
	"context"
	"time"

	"golang.org/x/sys/unix"
)

const (
	INTERVAL = 1 * time.Second
)

func WaitProcess(pid int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			err := unix.Kill(pid, 0)
			if err == unix.ESRCH {
				return true
			} else if err != nil {
				return false
			}
		}
	}
}
