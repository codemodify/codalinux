package applyexec

import (
	"fmt"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) btPower(op protocol.PlanOp) error {
	out, err := r.runHost("bluetoothctl", "power", onOff(op.Enabled))
	if err != nil {
		return fmt.Errorf("bluetoothctl power: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) btScan(op protocol.PlanOp) error {
	arg := onOff(op.Enabled)
	var out string
	var err error
	if arg == "on" {
		out, err = r.runHost("bluetoothctl", "--timeout", "8", "scan", "on")
	} else {
		out, err = r.runHost("bluetoothctl", "scan", "off")
	}
	if err != nil {
		return fmt.Errorf("bluetoothctl scan: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) btDevice(op protocol.PlanOp) error {
	addr := op.Device
	if addr == "" {
		addr = op.AddressBT
	}
	if err := checkBT(addr); err != nil {
		return err
	}
	sub := map[string]string{
		protocol.OpBTPair:       "pair",
		protocol.OpBTConnect:    "connect",
		protocol.OpBTDisconnect: "disconnect",
		protocol.OpBTTrust:      "trust",
	}[op.Type]
	if sub == "" {
		return fmt.Errorf("unknown bluetooth op")
	}
	out, err := r.runHost("bluetoothctl", sub, addr)
	if err != nil {
		return fmt.Errorf("bluetoothctl %s: %w (%s)", sub, err, strings.TrimSpace(out))
	}
	return nil
}
