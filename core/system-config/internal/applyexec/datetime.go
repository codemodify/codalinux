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

const timesyncdUnit = "systemd-timesyncd.service"

func (r *Runner) dateNTP(op protocol.PlanOp) error {
	out, err := r.runHost("timedatectl", "set-ntp", trueFalse(op.Enabled))
	if err == nil {
		return nil
	}
	td := strings.TrimSpace(out)
	// Live ISO has a writable overlay /etc, so timedatectl (EnableUnitFiles)
	// can create dbus-org.freedesktop.timesync1.service. Installed core-only
	// merges desktop /etc via systemd-confext or a read-only overlay: that
	// path is not writable, and set-ntp fails with
	// "File /etc/systemd/system/dbus-org.freedesktop.timesync1.service …".
	// Runtime enablement lives under /run/systemd/system, which systemd
	// honors and which stays writable on RO core + coda-data.
	if err2 := r.dateNTPRuntime(op.Enabled != nil && *op.Enabled); err2 == nil {
		return nil
	} else if ntpEtcBlocked(td) || ntpEtcBlocked(err2.Error()) {
		return fmt.Errorf("datetime.ntp: /etc immutable (RO core/sysext); %v; timedatectl: %s", err2, td)
	} else {
		return fmt.Errorf("timedatectl set-ntp: %w (%s); %v", err, td, err2)
	}
}

func (r *Runner) dateNTPRuntime(enable bool) error {
	if enable {
		_, _ = r.runHost("systemctl", "enable", "--runtime", timesyncdUnit)
		out, err := r.runHost("systemctl", "start", timesyncdUnit)
		if err != nil {
			return fmt.Errorf("systemctl start %s: %w (%s)", timesyncdUnit, err, strings.TrimSpace(out))
		}
		return nil
	}
	_, _ = r.runHost("systemctl", "disable", "--runtime", timesyncdUnit)
	out, err := r.runHost("systemctl", "stop", timesyncdUnit)
	if err != nil {
		return fmt.Errorf("systemctl stop %s: %w (%s)", timesyncdUnit, err, strings.TrimSpace(out))
	}
	return nil
}

func ntpEtcBlocked(s string) bool {
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "timesync1"),
		strings.Contains(low, "dbus-org.freedesktop.timesync"),
		strings.Contains(low, "read-only"),
		strings.Contains(low, "erofs"),
		strings.Contains(low, "/etc immutable"):
		return true
	case strings.Contains(low, "/etc/systemd/system") &&
		(strings.Contains(low, "file ") || strings.Contains(low, "symlink") || strings.Contains(low, "immutable")):
		return true
	default:
		return false
	}
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
