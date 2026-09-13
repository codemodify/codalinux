package udevwatch

import (
	"strings"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestParseUevent(t *testing.T) {
	raw := strings.Join([]string{
		"add@/devices/pci0000:00/0000:00:03.0/net/enp0s3",
		"ACTION=add",
		"DEVPATH=/devices/pci0000:00/0000:00:03.0/net/enp0s3",
		"SUBSYSTEM=net",
		"INTERFACE=enp0s3",
	}, "\x00")
	ev := Parse([]byte(raw))
	if ev.Action != "add" || ev.Subsystem != "net" || !strings.Contains(ev.Devpath, "enp0s3") {
		t.Fatalf("%+v", ev)
	}
	paths := PathsFor(ev)
	if len(paths) != 1 || paths[0] != protocol.PathNetwork {
		t.Fatalf("%v", paths)
	}
}

func TestPathsUSBAndDRM(t *testing.T) {
	usb := PathsFor(Event{Subsystem: "usb", Action: "add"})
	if len(usb) != 2 || usb[0] != protocol.PathDevicesUSB {
		t.Fatalf("%v", usb)
	}
	drm := PathsFor(Event{Subsystem: "drm"})
	if len(drm) == 0 || drm[0] != protocol.PathDisplay {
		t.Fatalf("%v", drm)
	}
	if PathsFor(Event{Subsystem: "block"}) != nil {
		t.Fatal("block should not map")
	}
}
