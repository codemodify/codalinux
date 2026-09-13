// Package applyexec is the closed allowlist for system-config-apply.
// No arbitrary shell. Display uses hyprctl eval hl.monitor({...}), not keyword.
package applyexec

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

type Runner struct {
	Hyprctl  string
	LookPath func(string) (string, error)
	Run      func(name string, args ...string) (string, error)
}

func New() *Runner {
	return &Runner{Hyprctl: "hyprctl"}
}

func (r *Runner) Exec(p protocol.Plan) error {
	for i, op := range p.Ops {
		if err := r.execOp(op); err != nil {
			return fmt.Errorf("op %d %s: %w", i, op.Type, err)
		}
	}
	return nil
}

func (r *Runner) execOp(op protocol.PlanOp) error {
	switch op.Type {
	case protocol.OpDisplayScale, protocol.OpDisplayMode:
		return r.hyprMonitor(op)
	default:
		return fmt.Errorf("refused: unknown op %q (allowlist: display.scale, display.mode)", op.Type)
	}
}

func (r *Runner) hyprMonitor(op protocol.PlanOp) error {
	if op.Output == "" {
		return fmt.Errorf("output required")
	}
	scale := op.Scale
	if scale <= 0 {
		scale = 1
	}
	mode := op.Mode
	if mode == "" {
		mode = "preferred"
	}
	expr := fmt.Sprintf("hl.monitor({ output = %s, mode = %s, position = %s, scale = %g })",
		luaString(op.Output), luaString(mode), luaString("auto"), scale)
	out, err := r.run(r.hyprctlBin(), "eval", expr)
	if err != nil {
		return fmt.Errorf("hyprctl eval: %w (%s)", err, strings.TrimSpace(out))
	}
	first := strings.TrimSpace(out)
	if i := strings.IndexByte(first, '\n'); i >= 0 {
		first = first[:i]
	}
	if first != "" && first != "ok" {
		return fmt.Errorf("hyprctl eval: %s", strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) hyprctlBin() string {
	if r.Hyprctl != "" {
		return r.Hyprctl
	}
	return "hyprctl"
}

func (r *Runner) run(name string, args ...string) (string, error) {
	if r.Run != nil {
		return r.Run(name, args...)
	}
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func luaString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// NeedHyprctl reports whether PATH has hyprctl (for diagnostics).
func NeedHyprctl() bool {
	_, err := exec.LookPath("hyprctl")
	return err == nil
}
