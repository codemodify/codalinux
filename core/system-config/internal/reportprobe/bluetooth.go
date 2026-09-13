package reportprobe

import (
	"encoding/json"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

func (p *Probe) bluetooth() (json.RawMessage, error) {
	m := protocol.BluetoothModel{}
	if raw, err := p.cmd("bluetoothctl", "show"); err == nil {
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
	}
	if raw, err := p.cmd("bluetoothctl", "devices"); err == nil {
		for _, line := range strings.Split(raw, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 || fields[0] != "Device" {
				continue
			}
			d := protocol.BTDevice{Address: fields[1], Name: strings.Join(fields[2:], " ")}
			if info, err := p.cmd("bluetoothctl", "info", d.Address); err == nil {
				for _, il := range strings.Split(info, "\n") {
					il = strings.TrimSpace(il)
					if strings.HasPrefix(il, "Paired:") {
						d.Paired = strings.Contains(strings.ToLower(il), "yes")
					}
					if strings.HasPrefix(il, "Connected:") {
						d.Connected = strings.Contains(strings.ToLower(il), "yes")
					}
					if strings.HasPrefix(il, "Trusted:") {
						d.Trusted = strings.Contains(strings.ToLower(il), "yes")
					}
				}
			}
			m.Devices = append(m.Devices, d)
		}
	}
	return rpc.Raw(m), nil
}
