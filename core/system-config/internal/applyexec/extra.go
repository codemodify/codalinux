package applyexec

import (
	"fmt"
	"os"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) printerDefault(op protocol.PlanOp) error {
	if err := checkPrinter(op.Name); err != nil {
		return err
	}
	out, err := r.runHost("lpadmin", "-d", op.Name)
	if err != nil {
		return fmt.Errorf("lpadmin -d: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) printerEnable(op protocol.PlanOp) error {
	if err := checkPrinter(op.Name); err != nil {
		return err
	}
	cmd := "cupsdisable"
	if op.Enabled != nil && *op.Enabled {
		cmd = "cupsenable"
	}
	out, err := r.runHost(cmd, op.Name)
	if err != nil {
		return fmt.Errorf("%s: %w (%s)", cmd, err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) userShell(op protocol.PlanOp) error {
	if err := checkUser(op.Name); err != nil {
		return err
	}
	if err := checkShell(op.Value); err != nil {
		return err
	}
	out, err := r.runHost("usermod", "-s", op.Value, op.Name)
	if err != nil {
		return fmt.Errorf("usermod -s: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) storageMount(op protocol.PlanOp) error {
	if err := checkBlock(op.Device); err != nil {
		return err
	}
	dev := "/dev/" + op.Device
	out, err := r.runHost("udisksctl", "mount", "-b", dev)
	if err != nil {
		return fmt.Errorf("udisksctl mount: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) storageUnmount(op protocol.PlanOp) error {
	if err := checkBlock(op.Device); err != nil {
		return err
	}
	mount := op.Value
	if mount == "" {
		mount = r.lookupMount(op.Device)
	}
	if systemMount(mount) {
		return fmt.Errorf("refused unmount of system path %q", mount)
	}
	dev := "/dev/" + op.Device
	out, err := r.runHost("udisksctl", "unmount", "-b", dev)
	if err != nil {
		return fmt.Errorf("udisksctl unmount: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) lookupMount(name string) string {
	if r.Run != nil {
		return ""
	}
	b, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return ""
	}
	want := "/dev/" + name
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == want || strings.HasSuffix(fields[0], "/"+name) {
			return fields[1]
		}
	}
	return ""
}

func systemMount(p string) bool {
	switch p {
	case "/", "/boot", "/usr", "/var", "/etc", "/home", "/opt":
		return true
	}
	return strings.HasPrefix(p, "/boot/") || strings.HasPrefix(p, "/usr/")
}
