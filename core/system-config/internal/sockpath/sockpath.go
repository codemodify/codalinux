// Package sockpath picks per-session Unix sockets (XDG runtime, else /tmp).
package sockpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	EnvD      = "CODA_SYSTEM_CONFIG_SOCKET"
	EnvApply  = "CODA_SYSTEM_CONFIG_APPLY_SOCKET"
	EnvReport = "CODA_SYSTEM_CONFIG_REPORT_SOCKET"
	EnvDir    = "CODA_SYSTEM_CONFIG_DIR"
)

func Dir() string {
	if d := os.Getenv(EnvDir); d != "" {
		return d
	}
	if x := os.Getenv("XDG_RUNTIME_DIR"); x != "" {
		return filepath.Join(x, "coda")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("coda-%d", os.Getuid()))
}

func Daemon() string {
	if s := os.Getenv(EnvD); s != "" {
		return s
	}
	return filepath.Join(Dir(), "system-configd.sock")
}

func Apply() string {
	if s := os.Getenv(EnvApply); s != "" {
		return s
	}
	return filepath.Join(Dir(), "system-config-apply.sock")
}

func Report() string {
	if s := os.Getenv(EnvReport); s != "" {
		return s
	}
	return filepath.Join(Dir(), "system-config-report.sock")
}

func EnsureDir(socket string) error {
	dir := filepath.Dir(socket)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return chownSeat(dir, 0o750)
}

// FixupSocket makes a listening Unix socket usable by the seat uid.
// Root apply creates the inode as root:root; D (unprivileged) must be able
// to connect. Mode 0660 + chown seat:seat when CODA_SYSTEM_CONFIG_UID is set.
func FixupSocket(socket string) error {
	return chownSeat(socket, 0o660)
}

func chownSeat(path string, mode os.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	uid := UID()
	if os.Getuid() != 0 || uid <= 0 {
		return nil
	}
	return os.Chown(path, uid, uid)
}

func UID() int {
	if s := os.Getenv("CODA_SYSTEM_CONFIG_UID"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return os.Getuid()
}
