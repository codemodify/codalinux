package applyexec

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) netIfaceEnable(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	state := "down"
	if op.Enabled != nil && *op.Enabled {
		state = "up"
	}
	out, err := r.runHost("ip", "link", "set", op.Device, state)
	if err != nil {
		return fmt.Errorf("ip link set: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) netIfaceMethod(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	if err := checkMethod(op.Method); err != nil {
		return err
	}
	if err := checkCIDR(op.Address); err != nil {
		return err
	}
	if err := checkIP(op.Gateway); err != nil {
		return err
	}
	for _, d := range op.DNS {
		if err := checkIP(d); err != nil {
			return err
		}
	}
	dir := r.NetworkDir
	if dir == "" {
		dir = "/etc/systemd/network"
	}
	body := networkdUnit(op)
	path := filepath.Join(dir, "20-coda-"+op.Device+".network")
	if err := r.writeFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	if out, err := r.runHost("networkctl", "reload"); err != nil {
		return fmt.Errorf("networkctl reload: %w (%s)", err, strings.TrimSpace(out))
	}
	if out, err := r.runHost("networkctl", "reconfigure", op.Device); err != nil {
		return fmt.Errorf("networkctl reconfigure: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func networkdUnit(op protocol.PlanOp) string {
	var b strings.Builder
	b.WriteString("[Match]\nName=")
	b.WriteString(op.Device)
	b.WriteString("\n\n[Network]\n")
	if op.Method == "static" {
		b.WriteString("DHCP=no\n")
		if op.Address != "" {
			b.WriteString("Address=")
			b.WriteString(op.Address)
			b.WriteString("\n")
		}
		if op.Gateway != "" {
			b.WriteString("Gateway=")
			b.WriteString(op.Gateway)
			b.WriteString("\n")
		}
		for _, d := range op.DNS {
			b.WriteString("DNS=")
			b.WriteString(d)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("DHCP=yes\n")
	}
	return b.String()
}

func (r *Runner) netWiFiConnect(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	if err := checkSSID(op.SSID); err != nil {
		return err
	}
	if err := checkPSK(op.PSK); err != nil {
		return err
	}
	var out string
	var err error
	if op.PSK != "" {
		out, err = r.runHost("iwctl", "--passphrase", op.PSK, "station", op.Device, "connect", op.SSID)
	} else {
		out, err = r.runHost("iwctl", "station", op.Device, "connect", op.SSID)
	}
	if err != nil {
		return fmt.Errorf("iwctl connect: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) netWiFiDisconnect(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	out, err := r.runHost("iwctl", "station", op.Device, "disconnect")
	if err != nil {
		return fmt.Errorf("iwctl disconnect: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}
