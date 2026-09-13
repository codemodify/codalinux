package applyexec

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) dateTimezone(op protocol.PlanOp) error {
	if err := checkTZ(op.Value); err != nil {
		return err
	}
	out, err := r.runHost("timedatectl", "set-timezone", op.Value)
	if err != nil {
		return fmt.Errorf("timedatectl set-timezone: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) dateNTP(op protocol.PlanOp) error {
	out, err := r.runHost("timedatectl", "set-ntp", trueFalse(op.Enabled))
	if err != nil {
		return fmt.Errorf("timedatectl set-ntp: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) dateTime(op protocol.PlanOp) error {
	if err := checkTime(op.Value); err != nil {
		return err
	}
	out, err := r.runHost("timedatectl", "set-time", op.Value)
	if err != nil {
		return fmt.Errorf("timedatectl set-time: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) localeLang(op protocol.PlanOp) error {
	if err := checkLang(op.Value); err != nil {
		return err
	}
	path := r.LocaleConf
	if path == "" {
		path = "/etc/locale.conf"
	}
	body := "LANG=" + op.Value + "\n"
	if err := r.writeFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	out, err := r.runHost("localectl", "set-locale", "LANG="+op.Value)
	if err != nil {
		return fmt.Errorf("localectl set-locale: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) sessionLock() error {
	out, err := r.runHost("loginctl", "lock-sessions")
	if err == nil {
		return nil
	}
	if out2, err2 := r.runSession("coda-hyprlock"); err2 == nil {
		return nil
	} else {
		out += " " + out2
	}
	return fmt.Errorf("loginctl lock-sessions: %w (%s)", err, strings.TrimSpace(out))
}

func (r *Runner) powerAction(action string) error {
	if action != "suspend" && action != "hibernate" {
		return fmt.Errorf("refused power action %q", action)
	}
	out, err := r.runHost("systemctl", action)
	if err != nil {
		return fmt.Errorf("systemctl %s: %w (%s)", action, err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) powerBrightness(op protocol.PlanOp) error {
	root := r.BacklightRoot
	if root == "" {
		root = "/sys/class/backlight"
	}
	name := op.ID
	if name == "" {
		name = op.Name
	}
	if name == "" {
		return fmt.Errorf("backlight device required")
	}
	if !reBacklight.MatchString(name) {
		return fmt.Errorf("invalid backlight")
	}
	n := int(op.Scale)
	if n < 0 {
		n = 0
	}
	path := filepath.Join(root, name, "brightness")
	return r.writeFile(path, []byte(fmt.Sprintf("%d\n", n)), 0o644)
}

func (r *Runner) powerLid(op protocol.PlanOp) error {
	if !reLid.MatchString(op.Value) {
		return fmt.Errorf("lid action must be ignore|suspend|lock|poweroff|hibernate")
	}
	path := r.LogindDrop
	if path == "" {
		path = "/etc/systemd/logind.conf.d/coda-lid.conf"
	}
	body := "[Login]\nHandleLidSwitch=" + op.Value + "\nHandleLidSwitchExternalPower=" + op.Value + "\n"
	if err := r.writeFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	out, err := r.runHost("systemctl", "kill", "-s", "HUP", "systemd-logind")
	if err != nil {
		return fmt.Errorf("reload logind: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}
