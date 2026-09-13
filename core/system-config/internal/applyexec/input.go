package applyexec

import (
	"fmt"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) inputKeymap(op protocol.PlanOp) error {
	if err := checkKeymap(op.Value); err != nil {
		return err
	}
	path := r.VConsole
	if path == "" {
		path = "/etc/vconsole.conf"
	}
	body := "KEYMAP=" + op.Value + "\n"
	if existing := r.readFile(path); existing != "" {
		var b strings.Builder
		replaced := false
		for _, line := range strings.Split(existing, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "KEYMAP=") {
				if !replaced {
					b.WriteString("KEYMAP=" + op.Value + "\n")
					replaced = true
				}
				continue
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			b.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				b.WriteString("\n")
			}
		}
		if !replaced {
			b.WriteString("KEYMAP=" + op.Value + "\n")
		}
		body = b.String()
	}
	_ = r.writeFile(path, []byte(body), 0o644)
	out, err := r.runHost("localectl", "set-keymap", op.Value)
	if err != nil {
		return fmt.Errorf("localectl set-keymap: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) inputKBLayout(op protocol.PlanOp) error {
	if err := checkKeymap(op.Value); err != nil {
		return err
	}
	if out, err := r.runHost("localectl", "set-x11-keymap", op.Value); err != nil {
		return fmt.Errorf("localectl set-x11-keymap: %w (%s)", err, strings.TrimSpace(out))
	}
	expr := fmt.Sprintf("hl.input({ kb_layout = %s })", luaString(op.Value))
	out, err := r.runHypr("eval", expr)
	if e := checkHyprOK(out, err); e != nil {
		return e
	}
	r.persistHypr(op)
	return nil
}

func (r *Runner) inputHypr(op protocol.PlanOp) error {
	var expr string
	switch op.Type {
	case protocol.OpInputPointerSpeed:
		s := op.Speed
		if s < -1 {
			s = -1
		}
		if s > 1 {
			s = 1
		}
		expr = fmt.Sprintf("hl.input({ sensitivity = %g })", s)
	case protocol.OpInputNaturalScroll:
		on := op.Enabled != nil && *op.Enabled
		expr = fmt.Sprintf("hl.input({ touchpad = { natural_scroll = %s } })", luaBool(on))
	case protocol.OpInputTapToClick:
		on := op.Enabled != nil && *op.Enabled
		expr = fmt.Sprintf("hl.input({ touchpad = { tap_to_click = %s } })", luaBool(on))
	default:
		return fmt.Errorf("unknown input op")
	}
	out, err := r.runHypr("eval", expr)
	if e := checkHyprOK(out, err); e != nil {
		return e
	}
	r.persistHypr(op)
	return nil
}
