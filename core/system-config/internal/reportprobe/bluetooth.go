package reportprobe

import (
	"encoding/json"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/bluez"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

func (p *Probe) bluetooth() (json.RawMessage, error) {
	if p.Run == nil {
		if m, ok := bluez.Collect(); ok {
			return rpc.Raw(m), nil
		}
	}
	// Never call bare `bluetoothctl` (interactive REPL). Always --timeout.
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
	trusted := map[string]bool{}
	if c, err := p.cmd("bluetoothctl", "--timeout", "2", "devices", "Connected"); err == nil {
		fillBTSet(c, connected)
	}
	if c, err := p.cmd("bluetoothctl", "--timeout", "2", "devices", "Paired"); err == nil {
		fillBTSet(c, paired)
	}
	if c, err := p.cmd("bluetoothctl", "--timeout", "2", "devices", "Trusted"); err == nil {
		fillBTSet(c, trusted)
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
			Trusted:   trusted[addr],
		})
	}
	return rpc.Raw(m), nil
}

func fillBTSet(raw string, dest map[string]bool) {
	for _, line := range strings.Split(raw, "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "Device" {
			dest[f[1]] = true
		}
	}
}
