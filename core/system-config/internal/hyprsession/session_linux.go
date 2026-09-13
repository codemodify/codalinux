//go:build linux

package hyprsession

import (
	"os"
	"os/exec"
	"syscall"
)

func applySessionCreds(cmd *exec.Cmd, s Session) {
	if c := sessionCreds(os.Geteuid(), s); c != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: c}
	}
}

func sessionCreds(euid int, s Session) *syscall.Credential {
	if euid != 0 || s.UID == 0 || uint32(euid) == s.UID {
		return nil
	}
	return &syscall.Credential{Uid: s.UID, Gid: s.GID, Groups: s.Groups}
}
