//go:build !linux

package udevwatch

import (
	"context"
	"fmt"
)

func Listen(ctx context.Context) (<-chan Event, error) {
	return nil, fmt.Errorf("NETLINK_KOBJECT_UEVENT is Linux-only")
}
