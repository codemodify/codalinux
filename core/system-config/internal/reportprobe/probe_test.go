package reportprobe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestDevicesFromSysfs(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "sys/class/dmi/id/sys_vendor"), "QEMU\n")
	mustWrite(t, filepath.Join(root, "sys/class/dmi/id/product_name"), "Standard PC\n")
	mustWrite(t, filepath.Join(root, "sys/bus/pci/devices/0000:00:00.0/vendor"), "0x8086\n")
	mustWrite(t, filepath.Join(root, "sys/bus/pci/devices/0000:00:00.0/device"), "0x1234\n")
	mustWrite(t, filepath.Join(root, "sys/bus/pci/devices/0000:00:00.0/class"), "0x060000\n")
	p := &Probe{Root: root}
	raw, err := p.Collect(protocol.PathDevicesSummary)
	if err != nil {
		t.Fatal(err)
	}
	var sum protocol.DevicesSummary
	if err := json.Unmarshal(raw, &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Vendor != "QEMU" || sum.PCICount != 1 {
		t.Fatalf("%+v", sum)
	}
	raw, err = p.Collect(protocol.PathDevicesPCI)
	if err != nil {
		t.Fatal(err)
	}
	var list protocol.PCIList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Devices) != 1 || list.Devices[0].Vendor != "0x8086" {
		t.Fatalf("%+v", list)
	}
}

func TestDisplayFromHyprctl(t *testing.T) {
	p := &Probe{Hyprctl: func() ([]byte, error) {
		return []byte(`[{"name":"Virtual-1","width":1920,"height":1080,"refreshRate":60,"scale":2,"focused":true}]`), nil
	}}
	raw, err := p.Collect(protocol.PathDisplay)
	if err != nil {
		t.Fatal(err)
	}
	var d protocol.DisplayModel
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Outputs) != 1 || d.Outputs[0].Scale != 2 || d.Outputs[0].Mode != "1920x1080@60" {
		t.Fatalf("%+v", d)
	}
}

func TestDisplayEmptyWithoutSession(t *testing.T) {
	p := &Probe{Discover: func() (hyprsession.Session, error) {
		return hyprsession.Session{}, errors.New("HYPRLAND_INSTANCE_SIGNATURE not set")
	}}
	raw, err := p.Collect(protocol.PathDisplay)
	if err != nil {
		t.Fatal(err)
	}
	var d protocol.DisplayModel
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Outputs) != 0 {
		t.Fatalf("%+v", d)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
