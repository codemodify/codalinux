//go:build linux

package udevwatch

import (
	"context"
	"fmt"

	"golang.org/x/sys/unix"
)

// Listen opens NETLINK_KOBJECT_UEVENT and emits parsed events until ctx is done.
// Unprivileged processes can usually join group 1 (kernel uevents). If that
// bind fails, Groups=0xffffffff is tried once.
func Listen(ctx context.Context) (<-chan Event, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, unix.NETLINK_KOBJECT_UEVENT)
	if err != nil {
		return nil, fmt.Errorf("netlink socket: %w", err)
	}
	sa := &unix.SockaddrNetlink{Family: unix.AF_NETLINK, Groups: 1}
	if err := unix.Bind(fd, sa); err != nil {
		sa.Groups = 0xffffffff
		if err2 := unix.Bind(fd, sa); err2 != nil {
			_ = unix.Close(fd)
			return nil, fmt.Errorf("netlink bind: %w", err)
		}
	}
	ch := make(chan Event, 16)
	go func() {
		defer close(ch)
		defer unix.Close(fd)
		go func() {
			<-ctx.Done()
			_ = unix.Shutdown(fd, unix.SHUT_RDWR)
		}()
		buf := make([]byte, 16<<10)
		for {
			if err := ctx.Err(); err != nil {
				return
			}
			n, _, err := unix.Recvfrom(fd, buf, 0)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if err == unix.EINTR {
					continue
				}
				return
			}
			if n <= 0 {
				continue
			}
			ev := Parse(buf[:n])
			if ev.Action == "" && ev.Subsystem == "" {
				continue
			}
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			default:
				// drop if the consumer is slow; next event or poll will catch up
			}
		}
	}()
	return ch, nil
}
