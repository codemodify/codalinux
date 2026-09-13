// Package udevwatch maps kernel kobject uevents to system-config paths.
package udevwatch

import (
	"bytes"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

// Event is one NETLINK_KOBJECT_UEVENT message.
type Event struct {
	Action    string
	Devpath   string
	Subsystem string
}

// Parse decodes a raw uevent datagram (NUL-separated KEY=VAL).
func Parse(b []byte) Event {
	var ev Event
	parts := bytes.Split(b, []byte{0})
	if len(parts) > 0 {
		head := string(parts[0])
		if i := strings.IndexByte(head, '@'); i >= 0 {
			ev.Action = head[:i]
			ev.Devpath = head[i+1:]
		}
	}
	for _, p := range parts[1:] {
		if len(p) == 0 {
			continue
		}
		k, v, ok := strings.Cut(string(p), "=")
		if !ok {
			continue
		}
		switch k {
		case "ACTION":
			ev.Action = v
		case "DEVPATH":
			ev.Devpath = v
		case "SUBSYSTEM":
			ev.Subsystem = v
		}
	}
	return ev
}

// PathsFor returns KnownPaths that should be re-probed for this event.
func PathsFor(e Event) []string {
	switch strings.ToLower(e.Subsystem) {
	case "net":
		return []string{protocol.PathNetwork}
	case "usb", "usbmisc":
		return []string{protocol.PathDevicesUSB, protocol.PathDevicesSummary}
	case "pci":
		return []string{protocol.PathDevicesPCI, protocol.PathDevicesSummary}
	case "bluetooth":
		return []string{protocol.PathBluetooth}
	case "input", "hid", "hidraw":
		return []string{protocol.PathInput}
	case "backlight", "power_supply", "leds":
		return []string{protocol.PathPower}
	case "dmi", "dmi-sysfs":
		return []string{protocol.PathHardwareDMI, protocol.PathDevicesSummary}
	case "drm", "graphics":
		return []string{protocol.PathDisplay, protocol.PathDevicesSummary}
	case "sound":
		return []string{protocol.PathAudio}
	case "block":
		return []string{protocol.PathStorage}
	default:
		if strings.Contains(e.Devpath, "/net/") {
			return []string{protocol.PathNetwork}
		}
		if strings.Contains(e.Devpath, "/usb") {
			return []string{protocol.PathDevicesUSB, protocol.PathDevicesSummary}
		}
		return nil
	}
}
