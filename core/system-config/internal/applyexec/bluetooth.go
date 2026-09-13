package applyexec

import (
	"fmt"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/bluez"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) useBlueZ() bool {
	// Injected Run is for tests / argv spies — stay on bluetoothctl.
	return r.Run == nil
}

func (r *Runner) btPower(op protocol.PlanOp) error {
	on := op.Enabled != nil && *op.Enabled
	if r.useBlueZ() {
		if err := bluez.SetPowered(on); err == nil {
			return nil
		}
	}
	out, err := r.runHost("bluetoothctl", "--timeout", "4", "power", onOff(op.Enabled))
	if err != nil {
		return fmt.Errorf("bluetoothctl power: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) btScan(op protocol.PlanOp) error {
	on := op.Enabled != nil && *op.Enabled
	if r.useBlueZ() {
		if err := bluez.SetDiscovering(on); err == nil {
			return nil
		}
	}
	arg := onOff(op.Enabled)
	var out string
	var err error
	if arg == "on" {
		out, err = r.runHost("bluetoothctl", "--timeout", "8", "scan", "on")
	} else {
		out, err = r.runHost("bluetoothctl", "--timeout", "2", "scan", "off")
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
	if err := checkPIN(op.PIN); err != nil {
		return err
	}
	if err := checkPIN(op.Value); err != nil {
		return err
	}
	if r.useBlueZ() {
		var err error
		switch op.Type {
		case protocol.OpBTPair:
			pin := op.PIN
			if pin == "" {
				pin = op.Value
			}
			err = bluez.Pair(addr, pin)
		case protocol.OpBTConnect:
			err = bluez.Connect(addr)
		case protocol.OpBTDisconnect:
			err = bluez.Disconnect(addr)
		case protocol.OpBTTrust:
			err = bluez.Trust(addr, true)
		}
		if err == nil {
			return nil
		}
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
	out, err := r.runHost("bluetoothctl", "--timeout", "10", sub, addr)
	if err != nil {
		return fmt.Errorf("bluetoothctl %s: %w (%s)", sub, err, strings.TrimSpace(out))
	}
	return nil
}
