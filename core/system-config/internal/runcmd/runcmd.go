// Package runcmd runs allowlisted binaries with a hard timeout.
// Never inherit an interactive REPL (bluetoothctl without --timeout).
package runcmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const Default = 2500 * time.Millisecond

func Run(d time.Duration, name string, args ...string) (string, error) {
	if d <= 0 {
		d = Default
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	return finish(ctx, d, name, cmd)
}

// Prepared runs an already-built command (env / credentials) with a timeout.
func Prepared(d time.Duration, cmd *exec.Cmd) (string, error) {
	if cmd == nil {
		return "", fmt.Errorf("nil command")
	}
	if d <= 0 {
		d = Default
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	name := cmd.Path
	if name == "" && len(cmd.Args) > 0 {
		name = cmd.Args[0]
	}
	args := []string{}
	if len(cmd.Args) > 1 {
		args = cmd.Args[1:]
	}
	ncmd := exec.CommandContext(ctx, name, args...)
	ncmd.Env = cmd.Env
	ncmd.Dir = cmd.Dir
	ncmd.SysProcAttr = cmd.SysProcAttr
	return finish(ctx, d, name, ncmd)
}

func finish(ctx context.Context, d time.Duration, name string, cmd *exec.Cmd) (string, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return buf.String(), fmt.Errorf("timeout after %s running %s", d, name)
	}
	return buf.String(), err
}
