//go:build !linux

package hyprsession

import "os/exec"

func applySessionCreds(*exec.Cmd, Session) {}
