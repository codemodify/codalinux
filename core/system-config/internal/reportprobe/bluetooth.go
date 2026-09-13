package reportprobe

import (
	"encoding/json"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

func (p *Probe) bluetooth() (json.RawMessage, error) {
	// Never call bare `bluetoothctl` (interactive REPL). Always --timeout.
	// On missing/stuck BlueZ return an empty adapter quickly — do not hang D.
	m := protocol.BluetoothModel{}
	raw, err := p.cmd("bluetoothctl", "--timeout", "2", "show")
	if err != nil {
		return rpc.Raw(m), nil
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Controller "):
			m.Adapter = strings.TrimPrefix(line, "Controller ")
			if i := strings.IndexByte(m.Adapter, ' '); i > 0 {
				m.Adapter = m.Adapter[:i]
			}
		case strings.HasPrefix(line, "Powered:"):
			m.Powered = strings.Contains(strings.ToLower(line), "yes")
		case strings.HasPrefix(line, "Discovering:"):
			m.Scanning = strings.Contains(strings.ToLower(line), "yes")
		}
	}
	devs, err := p.cmd("bluetoothctl", "--timeout", "2", "devices")
	if err != nil || strings.TrimSpace(devs) == "" {
		return rpc.Raw(m), nil
	}
	connected := map[string]bool{}
	paired := map[string]bool{}
	if c, err := p.cmd("bluetoothctl", "--timeout", "2", "devices", "Connected"); err == nil {
		for _, line := range strings.Split(c, "\n") {
			if f := strings.Fields(line); len(f) >= 2 && f[0] == "Device" {
				connected[f[1]] = true
			}
		}
	}
	if c, err := p.cmd("bluetoothctl", "--timeout", "2", "devices", "Paired"); err == nil {
		for _, line := range strings.Split(c, "\n") {
			if f := strings.Fields(line); len(f) >= 2 && f[0] == "Device" {
				paired[f[1]] = true
			}
		}
	}
	for _, line := range strings.Split(devs, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "Device" {
			continue
		}
		addr := fields[1]
		m.Devices = append(m.Devices, protocol.BTDevice{
			Address:   addr,
			Name:      strings.Join(fields[2:], " "),
			Connected: connected[addr],
			Paired:    paired[addr],
		})
	}
	return rpc.Raw(m), nil
}
