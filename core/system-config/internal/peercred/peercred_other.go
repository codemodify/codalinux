//go:build !linux

package peercred

import (
	"net"
	"os"
)

func UID(net.Conn) (int, error) { return os.Getuid(), nil }
