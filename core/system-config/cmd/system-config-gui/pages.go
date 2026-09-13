package main

import (
	"fmt"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/widgets"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (s *session) pageFor(path string) uitoolkit.Component {
	switch path {
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
	case protocol.PathDevicesSummary:
		return s.devicesPage()
	case protocol.PathSession:
		return s.sessionPage()
	case protocol.PathPower:
		return s.powerPage()
	default:
		return uitoolkit.NewLabel(path)
	}
}

func (s *session) refreshBtn(path string) *widgets.Button {
	return uitoolkit.NewButton("Refresh", func() {
		s.reload()
		s.rebuild()
		s.note("refreshed " + path)
	})
}

func (s *session) displayPage() uitoolkit.Component {
	rows := len(s.outputs)
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Output"}, {Title: "Mode", Width: 140}, {Title: "Scale", Width: 80},
	}, rows, func(row, col int) string {
		if row < 0 || row >= len(s.outputs) {
			return ""
		}
		o := s.outputs[row]
		switch col {
		case 1:
			return o.Mode
		case 2:
			return fmt.Sprintf("%g", o.Scale)
		default:
			if o.Name == s.outName {
				return o.Name + "  (focus)"
			}
			return o.Name
		}
	}, func(i int) {
		if i >= 0 && i < len(s.outputs) {
			s.outName = s.outputs[i].Name
			s.note("output " + s.outName)
		}
	})
	pct := float32(s.scale * 100)
	if pct < 100 {
		pct = 100
	}
	if pct > 200 {
		pct = 200
	}
	s.scaleLbl = uitoolkit.NewLabel(fmt.Sprintf("Scale  %.0f%%  (%.2f)", pct, s.scale))
	stage := func(v float64) {
		if v < 1 {
			v = 1
		}
		if v > 2 {
			v = 2
		}
		s.scale = v
		if s.scaleLbl != nil {
			s.scaleLbl.SetText(fmt.Sprintf("Scale  %.0f%%  (%.2f)", v*100, v))
		}
		if s.applyBtn != nil {
			s.applyBtn.SetEnabled(s.dirty() && s.cli != nil)
		}
	}
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Display"),
		uitoolkit.NewLabel("Stage a scale, then Apply. D runs hyprctl eval hl.monitor (not keyword)."),
		table, s.scaleLbl,
		uitoolkit.NewSlider(100, 200, pct, func(v float32) { stage(float64(v) / 100) }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Factor"), uitoolkit.NewNumberField(1, 2, s.scale, 0.25, stage)).WithGap(8),
		s.refreshBtn(protocol.PathDisplay),
	).WithGap(8)
}

func (s *session) networkPage() uitoolkit.Component {
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Link"}, {Title: "Type", Width: 70}, {Title: "State", Width: 80}, {Title: "Address"},
	}, len(s.net.Links), func(row, col int) string {
		if row < 0 || row >= len(s.net.Links) {
			return ""
		}
		l := s.net.Links[row]
		switch col {
		case 1:
			return l.Type
		case 2:
			return l.OperState
		case 3:
			if len(l.Addresses) > 0 {
				return l.Addresses[0]
			}
			return ""
		default:
			return l.Name
		}
	}, func(i int) {
		if i >= 0 && i < len(s.net.Links) {
			s.iface = s.net.Links[i].Name
			if s.net.Links[i].Type == "wlan" {
				s.wifiDev = s.net.Links[i].Name
			}
		}
	})
	ssid := uitoolkit.NewTextField(s.wifiSSID, "SSID", func(v string) { s.wifiSSID = v })
	psk := uitoolkit.NewTextField(s.wifiPSK, "PSK", func(v string) { s.wifiPSK = v })
	dev := uitoolkit.NewTextField(s.wifiDev, "wlan device", func(v string) { s.wifiDev = v })
	nets := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "SSID"}, {Title: "Security", Width: 90},
	}, len(s.net.WiFi.Networks), func(row, col int) string {
		if row < 0 || row >= len(s.net.WiFi.Networks) {
			return ""
		}
		n := s.net.WiFi.Networks[row]
		if col == 1 {
			return n.Security
		}
		return n.SSID
	}, func(i int) {
		if i >= 0 && i < len(s.net.WiFi.Networks) {
			s.wifiSSID = s.net.WiFi.Networks[i].SSID
			s.note("ssid " + s.wifiSSID)
		}
	})
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Network"),
		uitoolkit.NewLabel("systemd-networkd + iwd. No NetworkManager. Apply connect/disconnect and iface method."),
		table, nets,
		uitoolkit.NewRow(uitoolkit.NewLabel("Device"), dev).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("SSID"), ssid).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("PSK"), psk).WithGap(8),
		uitoolkit.NewSwitch("Hidden SSID", s.wifiHidden, func(on bool) { s.wifiHidden = on }),
		uitoolkit.NewButton("Disconnect staged", func() {
			s.net.WiFi.Disconnect = true
			s.wifiSSID = ""
			s.note("staged disconnect")
		}),
		s.refreshBtn(protocol.PathNetwork),
	).WithGap(8)
}

func (s *session) audioPage() uitoolkit.Component {
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Sink"}, {Title: "Vol", Width: 60}, {Title: "Mute", Width: 60},
	}, len(s.audio.Sinks), func(row, col int) string {
		if row < 0 || row >= len(s.audio.Sinks) {
			return ""
		}
		n := s.audio.Sinks[row]
		switch col {
		case 1:
			return fmt.Sprintf("%.0f%%", n.Volume*100)
		case 2:
			if n.Mute {
				return "yes"
			}
			return ""
		default:
			if n.Default {
				return n.ID + "  " + n.Name + " *"
			}
			return n.ID + "  " + n.Name
		}
	}, func(i int) {
		if i >= 0 && i < len(s.audio.Sinks) {
			s.audio.DefaultSink = s.audio.Sinks[i].ID
		}
	})
	s.volLbl = uitoolkit.NewLabel(fmt.Sprintf("Volume  %.0f%%", s.vol*100))
	defaults := fmt.Sprintf("Default sink %s   source %s", s.audio.DefaultSink, s.audio.DefaultSource)
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Audio"),
		uitoolkit.NewLabel("PipeWire via wpctl (session user). Default sink, volume, mute."),
		uitoolkit.NewLabel(defaults),
		table, s.volLbl,
		uitoolkit.NewSlider(0, 100, float32(s.vol*100), func(v float32) {
			s.vol = float64(v) / 100
			if s.volLbl != nil {
				s.volLbl.SetText(fmt.Sprintf("Volume  %.0f%%", s.vol*100))
			}
		}),
		uitoolkit.NewSwitch("Mute", s.mute, func(on bool) { s.mute = on }),
		s.refreshBtn(protocol.PathAudio),
	).WithGap(8)
}

func (s *session) bluetoothPage() uitoolkit.Component {
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Address"}, {Title: "Name"}, {Title: "State", Width: 120},
	}, len(s.bt.Devices), func(row, col int) string {
		if row < 0 || row >= len(s.bt.Devices) {
			return ""
		}
		d := s.bt.Devices[row]
		switch col {
		case 1:
			return d.Name
		case 2:
			st := ""
			if d.Connected {
				st = "connected"
			} else if d.Paired {
				st = "paired"
			}
			if d.Trusted {
				st += " trusted"
			}
			return st
		default:
			return d.Address
		}
	}, func(i int) {
		if i >= 0 && i < len(s.bt.Devices) {
			s.btAddr = s.bt.Devices[i].Address
		}
	})
	addr := uitoolkit.NewTextField(s.btAddr, "AA:BB:CC:DD:EE:FF", func(v string) { s.btAddr = v })
	stageAddr := func(kind string) {
		if s.btAddr == "" {
			s.note("pick a device first")
			return
		}
		switch kind {
		case "pair":
			s.btPair = []string{s.btAddr}
		case "connect":
			s.btConnect = []string{s.btAddr}
		case "disconnect":
			s.btDisconnect = []string{s.btAddr}
		case "trust":
			s.btTrust = []string{s.btAddr}
		}
		s.note("staged " + kind + " " + s.btAddr)
	}
	adapter := s.bt.Adapter
	if adapter == "" {
		adapter = "(none — BlueZ missing or stuck; refresh must not hang)"
	}
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Bluetooth"),
		uitoolkit.NewLabel("BlueZ via bluetoothctl --timeout. Power, scan, pair/connect/disconnect, trust."),
		uitoolkit.NewLabel("Adapter "+adapter),
		uitoolkit.NewSwitch("Adapter power", s.btPower, func(on bool) { s.btPower = on }),
		uitoolkit.NewSwitch("Scan", s.btScan, func(on bool) { s.btScan = on }),
		table,
		uitoolkit.NewRow(uitoolkit.NewLabel("Device"), addr).WithGap(8),
		uitoolkit.NewRow(
			uitoolkit.NewButton("Stage pair", func() { stageAddr("pair") }),
			uitoolkit.NewButton("Stage connect", func() { stageAddr("connect") }),
			uitoolkit.NewButton("Stage disconnect", func() { stageAddr("disconnect") }),
			uitoolkit.NewButton("Stage trust", func() { stageAddr("trust") }),
		).WithGap(8),
		s.refreshBtn(protocol.PathBluetooth),
	).WithGap(8)
}

func (s *session) inputPage() uitoolkit.Component {
	layout := uitoolkit.NewTextField(s.input.KBLayout, "us", func(v string) { s.input.KBLayout = v })
	keymap := uitoolkit.NewTextField(s.input.Keymap, "us", func(v string) { s.input.Keymap = v })
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Input"),
		uitoolkit.NewLabel("XKB / vconsole via localectl + Hyprland hl.input eval."),
		uitoolkit.NewRow(uitoolkit.NewLabel("KB layout"), layout).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Console keymap"), keymap).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Pointer speed"), uitoolkit.NewNumberField(-1, 1, s.input.PointerSpeed, 0.1, func(v float64) { s.input.PointerSpeed = v })).WithGap(8),
		uitoolkit.NewSwitch("Natural scroll", s.input.NaturalScroll, func(on bool) { s.input.NaturalScroll = on }),
		uitoolkit.NewSwitch("Tap to click", s.input.TapToClick, func(on bool) { s.input.TapToClick = on }),
		s.refreshBtn(protocol.PathInput),
	).WithGap(8)
}

func (s *session) datetimePage() uitoolkit.Component {
	tz := uitoolkit.NewTextField(s.dt.Timezone, "America/Denver", func(v string) { s.dt.Timezone = v })
	tm := uitoolkit.NewTextField(s.dt.Time, "YYYY-MM-DD HH:MM:SS", func(v string) { s.dt.Time = v })
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Date & time"),
		uitoolkit.NewLabel("timedatectl: timezone, NTP, optional manual time."),
		uitoolkit.NewRow(uitoolkit.NewLabel("Timezone"), tz).WithGap(8),
		uitoolkit.NewSwitch("NTP", s.dt.NTP, func(on bool) { s.dt.NTP = on }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Set time"), tm).WithGap(8),
		s.refreshBtn(protocol.PathDateTime),
	).WithGap(8)
}

func (s *session) localePage() uitoolkit.Component {
	lang := uitoolkit.NewTextField(s.loc.Lang, "en_US.UTF-8", func(v string) { s.loc.Lang = v })
	km := uitoolkit.NewTextField(s.loc.Keymap, "us", func(v string) { s.loc.Keymap = v })
	tz := uitoolkit.NewTextField(s.loc.Timezone, "America/Denver", func(v string) { s.loc.Timezone = v })
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Locale"),
		uitoolkit.NewLabel("localectl / locale.conf. Apply writes LANG and keymap."),
		uitoolkit.NewRow(uitoolkit.NewLabel("LANG"), lang).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Keymap"), km).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Timezone"), tz).WithGap(8),
		s.refreshBtn(protocol.PathLocale),
	).WithGap(8)
}

func (s *session) devicesPage() uitoolkit.Component {
	summary := fmt.Sprintf("%s %s   PCI %d   USB %d   DMI %s", s.summary.Vendor, s.summary.Product, s.summary.PCICount, s.summary.USBCount, s.dmi.Product)
	if s.pciTotal > pciCap {
		summary += fmt.Sprintf("   (PCI table %d of %d)", pciCap, s.pciTotal)
	}
	pci := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "PCI"}, {Title: "Vendor", Width: 80}, {Title: "Device", Width: 80}, {Title: "Class", Width: 80},
	}, len(s.pci), func(row, col int) string {
		if row < 0 || row >= len(s.pci) {
			return ""
		}
		d := s.pci[row]
		switch col {
		case 1:
			return d.Vendor
		case 2:
			return d.Device
		case 3:
			return d.Class
		default:
			return d.ID
		}
	}, nil)
	pci.Mono = true
	usb := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "USB"}, {Title: "Vendor", Width: 80}, {Title: "Product", Width: 80}, {Title: "Mfr"},
	}, len(s.usb), func(row, col int) string {
		if row < 0 || row >= len(s.usb) {
			return ""
		}
		d := s.usb[row]
		switch col {
		case 1:
			return d.Vendor
		case 2:
			return d.Product
		case 3:
			return d.Manufacturer
		default:
			return d.ID
		}
	}, nil)
	usb.Mono = true
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Devices"),
		uitoolkit.NewLabel(summary),
		pci, usb,
		s.refreshBtn("devices"),
	).WithGap(8)
}

func (s *session) sessionPage() uitoolkit.Component {
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "ID", Width: 50}, {Title: "User"}, {Title: "TTY", Width: 70}, {Title: "Type", Width: 80}, {Title: "State", Width: 80},
	}, len(s.sessions), func(row, col int) string {
		if row < 0 || row >= len(s.sessions) {
			return ""
		}
		x := s.sessions[row]
		switch col {
		case 1:
			return x.User
		case 2:
			return x.TTY
		case 3:
			return x.Type
		case 4:
			return x.State
		default:
			return x.ID
		}
	}, nil)
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Session"),
		uitoolkit.NewLabel("logind seats/users. Apply = lock-sessions only."),
		table,
		s.refreshBtn(protocol.PathSession),
	).WithGap(8)
}

func (s *session) powerPage() uitoolkit.Component {
	lid := uitoolkit.NewTextField(s.power.Lid, "ignore|suspend|lock", func(v string) { s.power.Lid = v })
	max := float32(s.power.MaxBrightness)
	if max <= 0 {
		max = 100
	}
	cur := float32(s.bright)
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Power"),
		uitoolkit.NewLabel("Brightness (sysfs), lid (logind drop-in), suspend/hibernate actions."),
		uitoolkit.NewLabel(fmt.Sprintf("CanSuspend %v   CanHibernate %v", s.power.CanSuspend, s.power.CanHibernate)),
		uitoolkit.NewLabel(fmt.Sprintf("Backlight %s  %d / %d", s.power.Backlight, int(s.bright), s.power.MaxBrightness)),
		uitoolkit.NewSlider(0, max, cur, func(v float32) { s.bright = float64(v) }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Lid"), lid).WithGap(8),
		uitoolkit.NewButton("Stage suspend", func() { s.power.Action = "suspend"; s.note("staged suspend") }),
		uitoolkit.NewButton("Stage hibernate", func() { s.power.Action = "hibernate"; s.note("staged hibernate") }),
		s.refreshBtn(protocol.PathPower),
	).WithGap(8)
}
