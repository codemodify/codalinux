package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldBool
	fieldFloat
	fieldAction
	fieldPick
)

type field struct {
	label  string
	kind   fieldKind
	get    func() string
	set    func(string)
	toggle func()
	act    func()
	hint   string
}

type pageView struct {
	title string
	help  string
	info  []string
	edits []field
}

func (s *session) page() pageView {
	switch s.path() {
	case protocol.PathDisplay:
		return s.displayPage()
	case protocol.PathNetwork:
		return s.networkPage()
	case protocol.PathAudio:
		return s.audioPage()
	case protocol.PathBluetooth:
		return s.bluetoothPage()
	case protocol.PathInput:
		return s.inputPage()
	case protocol.PathDateTime:
		return s.datetimePage()
	case protocol.PathLocale:
		return s.localePage()
	case protocol.PathSession:
		return s.sessionPage()
	case protocol.PathPower:
		return s.powerPage()
	case protocol.PathPrinters:
		return s.printersPage()
	case protocol.PathUsers:
		return s.usersPage()
	case protocol.PathStorage:
		return s.storagePage()
	case protocol.PathDevicesSummary:
		return s.devicesPage()
	case protocol.PathDevicesPCI:
		return s.pciPage()
	case protocol.PathDevicesUSB:
		return s.usbPage()
	case protocol.PathHardwareDMI:
		return s.dmiPage()
	default:
		return pageView{title: s.path(), help: "unknown path"}
	}
}

func (s *session) displayPage() pageView {
	info := []string{"Stage scale / mode / position, then Apply. D runs hyprctl eval hl.monitor."}
	for _, o := range s.outputs {
		mark := ""
		if o.Name == s.outName {
			mark = " *"
		}
		info = append(info, fmt.Sprintf("  %s  %s  pos=%s  scale=%g%s", o.Name, o.Mode, or(o.Position, fmt.Sprintf("%dx%d", o.X, o.Y)), o.Scale, mark))
	}
	if len(s.outputs) == 0 {
		info = append(info, "  (no outputs — refresh)")
	}
	return pageView{
		title: "Display",
		help:  "Apply writes display.scale / mode / position.",
		info:  info,
		edits: []field{
			textField("Output", func() string { return s.outName }, func(v string) { s.outName = v }),
			floatField("Scale", func() float64 { return s.scale }, func(v float64) { s.scale = clamp(v, 1, 2) }, "1.00–2.00"),
			textField("Mode", func() string { return s.outMode }, func(v string) { s.outMode = v }),
			textField("Position", func() string { return s.outPos }, func(v string) { s.outPos = v }),
		},
	}
}

func (s *session) networkPage() pageView {
	info := []string{"systemd-networkd + iwd. No NetworkManager."}
	for _, l := range s.net.Links {
		addr := ""
		if len(l.Addresses) > 0 {
			addr = l.Addresses[0]
		}
		info = append(info, fmt.Sprintf("  %s  %s  %s  %s", l.Name, l.Type, l.OperState, addr))
	}
	for _, n := range s.net.WiFi.Networks {
		info = append(info, fmt.Sprintf("  wifi %s  %s  %d", n.SSID, n.Security, n.Signal))
	}
	return pageView{
		title: "Network",
		help:  "Airplane / Wi-Fi connect / static IP. Apply is this section only.",
		info:  info,
		edits: []field{
			boolField("Airplane", func() bool { return s.airplane }, func(v bool) { s.airplane = v }),
			textField("Device", func() string { return s.wifiDev }, func(v string) { s.wifiDev = v }),
			textField("SSID", func() string { return s.wifiSSID }, func(v string) { s.wifiSSID = v }),
			textField("PSK", func() string { return s.wifiPSK }, func(v string) { s.wifiPSK = v }),
			boolField("Hidden SSID", func() bool { return s.wifiHidden }, func(v bool) { s.wifiHidden = v }),
			textField("Iface", func() string { return s.iface }, func(v string) { s.iface = v }),
			textField("Method", func() string { return s.netMethod }, func(v string) { s.netMethod = v }),
			textField("Address", func() string { return s.netAddr }, func(v string) { s.netAddr = v }),
			textField("Gateway", func() string { return s.netGW }, func(v string) { s.netGW = v }),
			textField("DNS", func() string { return s.netDNS }, func(v string) { s.netDNS = v }),
			textField("Search", func() string { return s.netSearch }, func(v string) { s.netSearch = v }),
			actionField("Stage disconnect", func() {
				s.net.WiFi.Disconnect = true
				s.wifiSSID = ""
			}),
		},
	}
}

func (s *session) audioPage() pageView {
	info := []string{
		fmt.Sprintf("Default sink %s   source %s", s.audio.DefaultSink, s.audio.DefaultSource),
		"PipeWire via pw-dump / wpctl. Persist to wireplumber 51-coda-defaults.conf.",
	}
	for _, n := range s.audio.Sinks {
		info = append(info, fmt.Sprintf("  sink %s  %s  vol=%.0f%% mute=%v", n.ID, n.Name, n.Volume*100, n.Mute))
	}
	for _, n := range s.audio.Sources {
		info = append(info, fmt.Sprintf("  source %s  %s", n.ID, n.Name))
	}
	return pageView{
		title: "Audio",
		help:  "Volume / mute / default sink and source.",
		info:  info,
		edits: []field{
			textField("Default sink", func() string { return s.audio.DefaultSink }, func(v string) { s.audio.DefaultSink = v }),
			textField("Default source", func() string { return s.audio.DefaultSource }, func(v string) { s.audio.DefaultSource = v }),
			floatField("Volume", func() float64 { return s.vol }, func(v float64) { s.vol = clamp(v, 0, 1) }, "0.00–1.00"),
			boolField("Mute", func() bool { return s.mute }, func(v bool) { s.mute = v }),
		},
	}
}

func (s *session) bluetoothPage() pageView {
	adapter := s.bt.Adapter
	if adapter == "" {
		adapter = "(none)"
	}
	info := []string{"Adapter " + adapter, "BlueZ D-Bus. Pair stages PIN then Apply."}
	for _, d := range s.bt.Devices {
		st := ""
		if d.Connected {
			st = "connected"
		} else if d.Paired {
			st = "paired"
		}
		if d.Trusted {
			st += " trusted"
		}
		info = append(info, fmt.Sprintf("  %s  %s  %s", d.Address, d.Name, st))
	}
	return pageView{
		title: "Bluetooth",
		help:  "Power / scan / pair / connect / trust. Apply this section only.",
		info:  info,
		edits: []field{
			boolField("Power", func() bool { return s.btPower }, func(v bool) { s.btPower = v }),
			boolField("Scan", func() bool { return s.btScan }, func(v bool) { s.btScan = v }),
			textField("Device", func() string { return s.btAddr }, func(v string) { s.btAddr = v }),
			textField("PIN", func() string { return s.btPIN }, func(v string) { s.btPIN = v }),
			actionField("Stage pair", func() {
				if s.btAddr != "" {
					s.btPair = []string{s.btAddr}
				}
			}),
			actionField("Stage connect", func() {
				if s.btAddr != "" {
					s.btConnect = []string{s.btAddr}
				}
			}),
			actionField("Stage disconnect", func() {
				if s.btAddr != "" {
					s.btDisconnect = []string{s.btAddr}
				}
			}),
			actionField("Stage trust", func() {
				if s.btAddr != "" {
					s.btTrust = []string{s.btAddr}
				}
			}),
		},
	}
}

func (s *session) inputPage() pageView {
	return pageView{
		title: "Input",
		help:  "XKB / vconsole + Hyprland hl.input.",
		info:  []string{"localectl + hypr persist."},
		edits: []field{
			textField("KB layout", func() string { return s.input.KBLayout }, func(v string) { s.input.KBLayout = v }),
			textField("Keymap", func() string { return s.input.Keymap }, func(v string) { s.input.Keymap = v }),
			floatField("Pointer speed", func() float64 { return s.input.PointerSpeed }, func(v float64) { s.input.PointerSpeed = clamp(v, -1, 1) }, "-1.00–1.00"),
			boolField("Natural scroll", func() bool { return s.input.NaturalScroll }, func(v bool) { s.input.NaturalScroll = v }),
			boolField("Tap to click", func() bool { return s.input.TapToClick }, func(v bool) { s.input.TapToClick = v }),
		},
	}
}

func (s *session) datetimePage() pageView {
	return pageView{
		title: "Date & time",
		help:  "timedatectl timezone / NTP / optional manual time.",
		info:  []string{fmt.Sprintf("RTC local=%v", s.dt.RTCLocal)},
		edits: []field{
			textField("Timezone", func() string { return s.dt.Timezone }, func(v string) { s.dt.Timezone = v }),
			boolField("NTP", func() bool { return s.dt.NTP }, func(v bool) { s.dt.NTP = v }),
			textField("Set time", func() string { return s.dt.Time }, func(v string) { s.dt.Time = v }),
		},
	}
}

func (s *session) localePage() pageView {
	return pageView{
		title: "Locale",
		help:  "localectl / locale.conf.",
		info:  nil,
		edits: []field{
			textField("LANG", func() string { return s.loc.Lang }, func(v string) { s.loc.Lang = v }),
			textField("Keymap", func() string { return s.loc.Keymap }, func(v string) { s.loc.Keymap = v }),
			textField("Timezone", func() string { return s.loc.Timezone }, func(v string) { s.loc.Timezone = v }),
		},
	}
}

func (s *session) sessionPage() pageView {
	info := []string{
		fmt.Sprintf("IdleHint %v  inhibits %d", s.idleHint, len(s.inhibits)),
		"Apply = lock only. Reboot/poweroff are not allowlisted.",
	}
	for _, x := range s.sessions {
		info = append(info, fmt.Sprintf("  sess %s  %s  %s  %s  %s", x.ID, x.User, x.TTY, x.Type, x.State))
	}
	for _, x := range s.seats {
		info = append(info, fmt.Sprintf("  seat %s  %v", x.ID, x.Sessions))
	}
	return pageView{
		title: "Session",
		help:  "logind. Stage lock, then Apply.",
		info:  info,
		edits: []field{
			actionField("Stage lock", func() { s.sessionAct = "lock" }),
		},
	}
}

func (s *session) powerPage() pageView {
	info := []string{
		fmt.Sprintf("CanSuspend %v  CanHibernate %v", s.power.CanSuspend, s.power.CanHibernate),
		fmt.Sprintf("Backlight %s  %d / %d", s.power.Backlight, int(s.bright), s.power.MaxBrightness),
		"Suspend/hibernate are gated. Reboot/poweroff are not allowlisted.",
	}
	return pageView{
		title: "Power",
		help:  "Brightness, lid, optional suspend/hibernate.",
		info:  info,
		edits: []field{
			floatField("Brightness", func() float64 { return s.bright }, func(v float64) { s.bright = v }, "0–max"),
			textField("Lid", func() string { return s.power.Lid }, func(v string) { s.power.Lid = v }),
			actionField("Stage suspend", func() { s.power.Action = "suspend" }),
			actionField("Stage hibernate", func() { s.power.Action = "hibernate" }),
		},
	}
}

func (s *session) printersPage() pageView {
	info := []string{"CUPS via lpstat / lpadmin. present=false when cups is missing."}
	if s.printers.Default != "" {
		info = append(info, "Default: "+s.printers.Default)
	}
	for _, p := range s.printers.Printers {
		info = append(info, fmt.Sprintf("  %s  %s", p.Name, p.State))
	}
	return pageView{
		title: "Printers",
		help:  "Apply sets default and enable.",
		info:  info,
		edits: []field{
			textField("Default", func() string { return s.printerName }, func(v string) { s.printerName = v }),
			boolField("Enabled", func() bool { return s.printerOn }, func(v bool) { s.printerOn = v }),
		},
	}
}

func (s *session) usersPage() pageView {
	info := []string{"Local accounts from /etc/passwd. Apply changes login shell only."}
	for _, u := range s.users.Users {
		info = append(info, fmt.Sprintf("  %s  uid=%d  %s  %s", u.Name, u.UID, u.Home, u.Shell))
	}
	return pageView{
		title: "Users",
		help:  "usermod -s. No add/delete/password.",
		info:  info,
		edits: []field{
			textField("User", func() string { return s.userName }, func(v string) { s.userName = v }),
			textField("Shell", func() string { return s.userShell }, func(v string) { s.userShell = v }),
		},
	}
}

func (s *session) storagePage() pageView {
	info := []string{"lsblk / udisks. System mounts (/, /boot, /usr, /home) are refused."}
	for _, b := range s.storage.Block {
		info = append(info, fmt.Sprintf("  %s  %s  %s  %s  %s", b.Name, b.Type, b.Size, b.Mount, b.Model))
	}
	return pageView{
		title: "Storage",
		help:  "Stage mount or unmount, then Apply.",
		info:  info,
		edits: []field{
			textField("Device", func() string { return s.storageName }, func(v string) { s.storageName = v }),
			actionField("Stage mount", func() { s.storageAct = "mount" }),
			actionField("Stage unmount", func() { s.storageAct = "unmount" }),
		},
	}
}

func (s *session) devicesPage() pageView {
	info := []string{
		fmt.Sprintf("%s %s   PCI %d   USB %d   DMI %s", s.summary.Vendor, s.summary.Product, s.summary.PCICount, s.summary.USBCount, s.dmi.Product),
		"Observe-only. Refresh reloads summary + PCI/USB/DMI.",
	}
	return pageView{title: "Devices", help: "observe-only", info: info}
}

func (s *session) pciPage() pageView {
	info := []string{fmt.Sprintf("%d PCI devices (observe-only)", len(s.pci))}
	for i, d := range s.pci {
		if i >= 40 {
			info = append(info, "  …")
			break
		}
		info = append(info, fmt.Sprintf("  %s  %s  %s  %s", d.ID, d.Vendor, d.Device, d.Class))
	}
	return pageView{title: "PCI", help: "observe-only", info: info}
}

func (s *session) usbPage() pageView {
	info := []string{fmt.Sprintf("%d USB devices (observe-only)", len(s.usb))}
	for i, d := range s.usb {
		if i >= 40 {
			info = append(info, "  …")
			break
		}
		info = append(info, fmt.Sprintf("  %s  %s  %s  %s", d.ID, d.Vendor, d.Product, d.Manufacturer))
	}
	return pageView{title: "USB", help: "observe-only", info: info}
}

func (s *session) dmiPage() pageView {
	return pageView{
		title: "DMI",
		help:  "observe-only",
		info: []string{
			"Vendor " + s.dmi.Vendor,
			"Product " + s.dmi.Product,
			"Board " + s.dmi.Board,
			"BIOS " + s.dmi.BiosVersion + "  " + s.dmi.BiosDate,
			"Chassis " + s.dmi.Chassis,
		},
	}
}

func textField(label string, get func() string, set func(string)) field {
	return field{label: label, kind: fieldText, get: get, set: set, hint: "Enter to edit"}
}

func boolField(label string, get func() bool, set func(bool)) field {
	return field{
		label: label, kind: fieldBool,
		get: func() string {
			if get() {
				return "on"
			}
			return "off"
		},
		toggle: func() { set(!get()) },
		hint:   "Space to toggle",
	}
}

func floatField(label string, get func() float64, set func(float64), hint string) field {
	return field{
		label: label, kind: fieldFloat,
		get: func() string { return fmt.Sprintf("%.2f", get()) },
		set: func(v string) {
			n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err == nil {
				set(n)
			}
		},
		hint: hint,
	}
}

func actionField(label string, act func()) field {
	return field{label: label, kind: fieldAction, get: func() string { return "Enter" }, act: act, hint: "Enter to stage"}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
