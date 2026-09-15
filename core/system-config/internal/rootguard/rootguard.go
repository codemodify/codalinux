// Package rootguard refuses system-config user daemons as uid 0.
// Root would bind /run/user/0 and duplicate the seat-user control plane.
package rootguard

import (
	"fmt"
	"os"
)

// Allowed reports whether this process may start a user-session daemon.
func Allowed() bool {
	if os.Getuid() != 0 {
		return true
	}
	return os.Getenv("CODA_SYSTEM_CONFIG_ALLOW_ROOT") == "1"
}

// RefuseRoot exits the process if it would run as root, unless
// CODA_SYSTEM_CONFIG_ALLOW_ROOT=1 (tests / recovery only).
func RefuseRoot(name string) {
	if Allowed() {
		return
	}
	fmt.Fprintf(os.Stderr, "%s refuses to run as root (would bind /run/user/0). Start as the seat user.\n", name)
	os.Exit(2)
}
