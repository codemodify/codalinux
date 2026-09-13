package applyexec

import (
	"fmt"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) audioID(op protocol.PlanOp) (string, error) {
	id := op.ID
	if id == "" {
		id = op.Name
	}
	if id == "" {
		id = "@DEFAULT_AUDIO_SINK@"
	}
	if err := checkNodeID(id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *Runner) audioDefault(op protocol.PlanOp) error {
	id, err := r.audioID(op)
	if err != nil {
		return err
	}
	out, err := r.runSession("wpctl", "set-default", id)
	if err != nil {
		return fmt.Errorf("wpctl set-default: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) audioVolume(op protocol.PlanOp) error {
	id, err := r.audioID(op)
	if err != nil {
		return err
	}
	v := op.Volume
	if v < 0 {
		v = 0
	}
	if v > 1.5 {
		v = 1.5
	}
	arg := fmt.Sprintf("%g", v)
	out, err := r.runSession("wpctl", "set-volume", id, arg)
	if err != nil {
		return fmt.Errorf("wpctl set-volume: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) audioMute(op protocol.PlanOp) error {
	id, err := r.audioID(op)
	if err != nil {
		return err
	}
	flag := "0"
	if op.Mute != nil && *op.Mute {
		flag = "1"
	}
	out, err := r.runSession("wpctl", "set-mute", id, flag)
	if err != nil {
		return fmt.Errorf("wpctl set-mute: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}
