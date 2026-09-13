package reportprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

func (p *Probe) input() (json.RawMessage, error) {
	m := protocol.InputModel{
		Keymap:        keymap(p.root("etc/vconsole.conf")),
		KBLayout:      "us",
		NaturalScroll: true,
		TapToClick:    true,
	}
	if raw, err := p.cmd("localectl", "status"); err == nil {
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "VC Keymap:") {
				m.Keymap = strings.TrimSpace(strings.TrimPrefix(line, "VC Keymap:"))
			}
			if strings.HasPrefix(line, "X11 Layout:") {
				m.KBLayout = strings.TrimSpace(strings.TrimPrefix(line, "X11 Layout:"))
			}
		}
	}
	if m.KBLayout == "" {
		m.KBLayout = m.Keymap
	}
	return rpc.Raw(m), nil
}

func (p *Probe) datetime() (json.RawMessage, error) {
	m := protocol.DateTimeModel{
		Timezone: timezone(p.root("etc/localtime")),
	}
	if raw, err := p.cmd("timedatectl", "show", "--no-pager"); err == nil {
		for _, line := range strings.Split(raw, "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok {
				continue
			}
			switch k {
			case "Timezone":
				m.Timezone = v
			case "NTP", "NTPSynchronized":
				if k == "NTP" {
					m.NTP = v == "yes"
				}
			case "LocalRTC":
				m.RTCLocal = v == "yes"
			case "TimeUSec":
				m.Time = v
			}
		}
	}
	return rpc.Raw(m), nil
}

func (p *Probe) session() (json.RawMessage, error) {
	m := protocol.SessionModel{}
	raw, err := p.cmd("loginctl", "list-sessions", "--no-legend", "--no-pager")
	if err != nil {
		return rpc.Raw(m), nil
	}
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		s := protocol.LoginSession{ID: fields[0], User: fields[2]}
		if n, err := strconv.Atoi(fields[1]); err == nil {
			s.UID = n
		}
		if len(fields) > 3 {
			s.Seat = fields[3]
		}
		if len(fields) > 4 {
			s.TTY = fields[4]
		}
		if info, err := p.cmd("loginctl", "show-session", s.ID, "-p", "Type", "-p", "Class", "-p", "State"); err == nil {
			for _, il := range strings.Split(info, "\n") {
				k, v, ok := strings.Cut(strings.TrimSpace(il), "=")
				if !ok {
					continue
				}
				switch k {
				case "Type":
					s.Type = v
				case "Class":
					s.Class = v
				case "State":
					s.State = v
				}
			}
		}
		m.Sessions = append(m.Sessions, s)
	}
	return rpc.Raw(m), nil
}

func (p *Probe) power() (json.RawMessage, error) {
	m := protocol.PowerModel{}
	if raw, err := p.cmd("busctl", "get-property", "org.freedesktop.login1", "/org/freedesktop/login1",
		"org.freedesktop.login1.Manager", "CanSuspend"); err == nil && strings.Contains(raw, "yes") {
		m.CanSuspend = true
	}
	if raw, err := p.cmd("busctl", "get-property", "org.freedesktop.login1", "/org/freedesktop/login1",
		"org.freedesktop.login1.Manager", "CanHibernate"); err == nil && strings.Contains(raw, "yes") {
		m.CanHibernate = true
	}
	lid := readLogind(p.root("etc/systemd/logind.conf"), "HandleLidSwitch")
	m.Lid = lid
	dir := p.root("sys/class/backlight")
	ents, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range ents {
			m.Backlight = e.Name()
			if n, err := strconv.Atoi(readTrim(filepath.Join(dir, e.Name(), "brightness"))); err == nil {
				m.Brightness = n
			}
			if n, err := strconv.Atoi(readTrim(filepath.Join(dir, e.Name(), "max_brightness"))); err == nil {
				m.MaxBrightness = n
			}
			break
		}
	}
	return rpc.Raw(m), nil
}

func readLogind(path, key string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (p *Probe) devicesUSB() (json.RawMessage, error) {
	dir := p.root("sys/bus/usb/devices")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return rpc.Raw(protocol.USBList{}), nil
	}
	var list protocol.USBList
	for _, e := range ents {
		if strings.Contains(e.Name(), ":") {
			continue
		}
		dev := filepath.Join(dir, e.Name())
		list.Devices = append(list.Devices, protocol.USBDevice{
			ID:           e.Name(),
			Vendor:       readTrim(filepath.Join(dev, "idVendor")),
			Product:      readTrim(filepath.Join(dev, "idProduct")),
			Manufacturer: readTrim(filepath.Join(dev, "manufacturer")),
		})
	}
	return rpc.Raw(list), nil
}

func (p *Probe) hardwareDMI() (json.RawMessage, error) {
	d := protocol.DMI{
		Vendor:      readTrim(p.root("sys/class/dmi/id/sys_vendor")),
		Product:     readTrim(p.root("sys/class/dmi/id/product_name")),
		Version:     readTrim(p.root("sys/class/dmi/id/product_version")),
		Board:       readTrim(p.root("sys/class/dmi/id/board_name")),
		BiosDate:    readTrim(p.root("sys/class/dmi/id/bios_date")),
		BiosVersion: readTrim(p.root("sys/class/dmi/id/bios_version")),
		Chassis:     readTrim(p.root("sys/class/dmi/id/chassis_type")),
	}
	return rpc.Raw(d), nil
}
