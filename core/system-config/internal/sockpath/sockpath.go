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
	return os.MkdirAll(filepath.Dir(socket), 0o700)
}

func UID() int {
	if s := os.Getenv("CODA_SYSTEM_CONFIG_UID"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return os.Getuid()
}
