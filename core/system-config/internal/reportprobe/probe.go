// Package reportprobe collects observed state. It never applies config.
package reportprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

type Probe struct {
	Root     string // default /
	Hyprctl  func() ([]byte, error)
	Discover func() (hyprsession.Session, error)
}

func New() *Probe { return &Probe{Root: "/"} }

func (p *Probe) Collect(path string) (json.RawMessage, error) {
	path = protocol.NormalizePath(path)
	switch path {
	case protocol.PathDisplay:
		return p.display()
	case protocol.PathDevicesSummary:
		return p.devicesSummary()
	case protocol.PathDevicesPCI:
		return p.devicesPCI()
	case protocol.PathLocale:
		return p.locale()
	case "", protocol.PathSubmodels:
		return rpc.Raw(map[string]any{"paths": protocol.StarterPaths}), nil
	default:
		return nil, errStr("unknown path " + path)
	}
}

func (p *Probe) root(elem ...string) string {
	base := p.Root
	if base == "" {
		base = "/"
	}
	return filepath.Join(append([]string{base}, elem...)...)
}

func (p *Probe) display() (json.RawMessage, error) {
	raw, err := p.hyprJSON()
	if err != nil || len(raw) == 0 {
		return rpc.Raw(protocol.DisplayModel{}), nil
	}
	var mons []struct {
		Name        string  `json:"name"`
		Width       int     `json:"width"`
		Height      int     `json:"height"`
		RefreshRate float64 `json:"refreshRate"`
		Scale       float64 `json:"scale"`
		Focused     bool    `json:"focused"`
	}
	if err := json.Unmarshal(raw, &mons); err != nil {
		return rpc.Raw(protocol.DisplayModel{}), nil
	}
	out := protocol.DisplayModel{}
	for _, m := range mons {
		hz := int(m.RefreshRate + 0.5)
		if hz == 0 {
			hz = 60
		}
		out.Outputs = append(out.Outputs, protocol.Output{
			Name: m.Name, Width: m.Width, Height: m.Height,
			RefreshHz: hz, Scale: m.Scale, Focused: m.Focused,
			Mode: formatMode(m.Width, m.Height, hz),
		})
	}
	return rpc.Raw(out), nil
}

func (p *Probe) hyprJSON() ([]byte, error) {
	if p.Hyprctl != nil {
		return p.Hyprctl()
	}
	discover := p.Discover
	if discover == nil {
		discover = hyprsession.Discover
	}
	sess, err := discover()
	if err != nil {
		return nil, err
	}
	cmd := sess.Command("hyprctl", "-j", "monitors")
	return cmd.Output()
}

func formatMode(w, h, hz int) string {
	if w == 0 || h == 0 {
		return ""
	}
	return strconv.Itoa(w) + "x" + strconv.Itoa(h) + "@" + strconv.Itoa(hz)
}

func (p *Probe) devicesSummary() (json.RawMessage, error) {
	s := protocol.DevicesSummary{
		Vendor:   readTrim(p.root("sys/class/dmi/id/sys_vendor")),
		Product:  readTrim(p.root("sys/class/dmi/id/product_name")),
		Board:    readTrim(p.root("sys/class/dmi/id/board_name")),
		PCICount: countDir(p.root("sys/bus/pci/devices")),
		USBCount: countUSB(p.root("sys/bus/usb/devices")),
	}
	return rpc.Raw(s), nil
}

func (p *Probe) devicesPCI() (json.RawMessage, error) {
	dir := p.root("sys/bus/pci/devices")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return rpc.Raw(protocol.PCIList{}), nil
	}
	var list protocol.PCIList
	for _, e := range ents {
		id := e.Name()
		dev := filepath.Join(dir, id)
		list.Devices = append(list.Devices, protocol.PCIDevice{
			ID:     id,
			Vendor: readTrim(filepath.Join(dev, "vendor")),
			Device: readTrim(filepath.Join(dev, "device")),
			Class:  readTrim(filepath.Join(dev, "class")),
		})
	}
	return rpc.Raw(list), nil
}

func (p *Probe) locale() (json.RawMessage, error) {
	loc := protocol.Locale{
		Lang:     localeLang(p.root("etc/locale.conf")),
		Timezone: timezone(p.root("etc/localtime")),
		Keymap:   keymap(p.root("etc/vconsole.conf")),
	}
	if loc.Lang == "" {
		loc.Lang = os.Getenv("LANG")
	}
	return rpc.Raw(loc), nil
}

func localeLang(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LANG=") {
			return strings.Trim(strings.TrimPrefix(line, "LANG="), `"'`)
		}
	}
	return ""
}

func keymap(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "KEYMAP=") {
			return strings.Trim(strings.TrimPrefix(line, "KEYMAP="), `"'`)
		}
	}
	return ""
}

func timezone(localtime string) string {
	target, err := os.Readlink(localtime)
	if err != nil {
		return ""
	}
	const zone = "zoneinfo/"
	if i := strings.LastIndex(target, zone); i >= 0 {
		return target[i+len(zone):]
	}
	return target
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func countDir(path string) int {
	ents, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range ents {
		if e.Name() != "." && e.Name() != ".." {
			n++
		}
	}
	return n
}

func countUSB(path string) int {
	ents, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range ents {
		// skip usb hub interface nodes like 1-0:1.0
		if strings.Contains(e.Name(), ":") {
			continue
		}
		n++
	}
	return n
}

type errStr string

func (e errStr) Error() string { return string(e) }
