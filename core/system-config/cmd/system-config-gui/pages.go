package main

import (
	"fmt"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/widget"
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
	case protocol.PathPrinters:
		return s.printersPage()
	case protocol.PathUsers:
		return s.usersPage()
	case protocol.PathStorage:
		return s.storagePage()
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
		{Title: "Output"}, {Title: "Mode", Width: 140}, {Title: "Pos", Width: 90}, {Title: "Scale", Width: 80},
	}, rows, func(row, col int) string {
		if row < 0 || row >= len(s.outputs) {
			return ""
		}
		o := s.outputs[row]
		switch col {
		case 1:
			return o.Mode
		case 2:
			if o.Position != "" {
				return o.Position
			}
			return fmt.Sprintf("%dx%d", o.X, o.Y)
		case 3:
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
			s.outMode = s.outputs[i].Mode
			s.outPos = s.outputs[i].Position
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
		uitoolkit.NewLabel("Stage scale / mode / position, then Apply. D runs hyprctl eval hl.monitor (not keyword)."),
		table, s.scaleLbl,
		uitoolkit.NewSlider(100, 200, pct, func(v float32) { stage(float64(v) / 100) }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Factor"), uitoolkit.NewNumberField(1, 2, s.scale, 0.25, stage)).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Mode"), uitoolkit.NewTextField(s.outMode, "1920x1080@60", func(v string) { s.outMode = v })).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Position"), uitoolkit.NewTextField(s.outPos, "auto or 0x0", func(v string) { s.outPos = v })).WithGap(8),
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
		{Title: "SSID"}, {Title: "Security", Width: 90}, {Title: "Signal", Width: 70},
	}, len(s.net.WiFi.Networks), func(row, col int) string {
		if row < 0 || row >= len(s.net.WiFi.Networks) {
			return ""
		}
		n := s.net.WiFi.Networks[row]
		switch col {
		case 1:
			return n.Security
		case 2:
			if n.Signal == 0 {
				return ""
			}
			return fmt.Sprintf("%d", n.Signal)
		default:
			return n.SSID
		}
	}, func(i int) {
		if i >= 0 && i < len(s.net.WiFi.Networks) {
			s.wifiSSID = s.net.WiFi.Networks[i].SSID
			s.note("ssid " + s.wifiSSID)
		}
	})
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Network"),
		uitoolkit.NewLabel("systemd-networkd + iwd. Static IP writes a drop-in or 20-coda-*.network (survives reboot). No NetworkManager."),
		table, nets,
		uitoolkit.NewSwitch("Airplane mode (rfkill)", s.airplane, func(on bool) { s.airplane = on }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Device"), dev).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("SSID"), ssid).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("PSK"), psk).WithGap(8),
		uitoolkit.NewSwitch("Hidden SSID", s.wifiHidden, func(on bool) { s.wifiHidden = on }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Method"), uitoolkit.NewTextField(s.netMethod, "dhcp|static", func(v string) { s.netMethod = v })).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Address"), uitoolkit.NewTextField(s.netAddr, "10.0.2.15/24", func(v string) { s.netAddr = v })).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Gateway"), uitoolkit.NewTextField(s.netGW, "10.0.2.2", func(v string) { s.netGW = v })).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("DNS"), uitoolkit.NewTextField(s.netDNS, "1.1.1.1 8.8.8.8", func(v string) { s.netDNS = v })).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Search"), uitoolkit.NewTextField(s.netSearch, "example.lan", func(v string) { s.netSearch = v })).WithGap(8),
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
		{Title: "Sink (full name)"}, {Title: "Vol", Width: 60}, {Title: "Mute", Width: 60},
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
			s.vol = s.audio.Sinks[i].Volume
			s.mute = s.audio.Sinks[i].Mute
		}
	})
	sources := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Source (full name)"}, {Title: "Vol", Width: 60}, {Title: "Mute", Width: 60},
	}, len(s.audio.Sources), func(row, col int) string {
		if row < 0 || row >= len(s.audio.Sources) {
			return ""
		}
		n := s.audio.Sources[row]
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
		if i >= 0 && i < len(s.audio.Sources) {
			s.audio.DefaultSource = s.audio.Sources[i].ID
		}
	})
	s.volLbl = uitoolkit.NewLabel(fmt.Sprintf("Volume  %.0f%%", s.vol*100))
	defaults := fmt.Sprintf("Default sink %s   source %s", s.audio.DefaultSink, s.audio.DefaultSource)
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Audio"),
		uitoolkit.NewLabel("PipeWire via pw-dump / wpctl. Default sink/source persist to ~/.config/wireplumber/wireplumber.conf.d/51-coda-defaults.conf."),
		uitoolkit.NewLabel(defaults),
		table, sources, s.volLbl,
		uitoolkit.NewSlider(0, 100, float32(s.vol*100), func(v float32) {
			s.vol = float64(v) / 100
			if s.volLbl != nil {
				s.volLbl.SetText(fmt.Sprintf("Volume  %.0f%%", s.vol*100))
			}
			for i := range s.audio.Sinks {
				if s.audio.Sinks[i].ID == s.audio.DefaultSink || s.audio.Sinks[i].Default {
					s.audio.Sinks[i].Volume = s.vol
				}
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
			s.showPINDialog()
			return
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
		uitoolkit.NewLabel("BlueZ D-Bus pairing agent. Pair opens a PIN dialog; Apply writes $XDG_RUNTIME_DIR/coda/bluetooth-pin and the agent reads it (or waits up to 12s)."),
		uitoolkit.NewLabel("Adapter "+adapter),
		uitoolkit.NewSwitch("Adapter power", s.btPower, func(on bool) { s.btPower = on }),
		uitoolkit.NewSwitch("Scan", s.btScan, func(on bool) { s.btScan = on }),
		table,
		uitoolkit.NewRow(uitoolkit.NewLabel("Device"), addr).WithGap(8),
		uitoolkit.NewRow(
			uitoolkit.NewButton("Pair…", func() { stageAddr("pair") }),
			uitoolkit.NewButton("Stage connect", func() { stageAddr("connect") }),
			uitoolkit.NewButton("Stage disconnect", func() { stageAddr("disconnect") }),
			uitoolkit.NewButton("Stage trust", func() { stageAddr("trust") }),
		).WithGap(8),
		s.refreshBtn(protocol.PathBluetooth),
	).WithGap(8)
}

func (s *session) showPINDialog() {
	if s.btAddr == "" {
		s.note("pick a device first")
		return
	}
	host := widget.Component(nil)
	if s.win != nil {
		host = s.win.Content()
	}
	if host == nil {
		s.btPair = []string{s.btAddr}
		s.note("staged pair " + s.btAddr + " (no window for PIN dialog)")
		return
	}
	pin := s.btPIN
	field := uitoolkit.NewPasswordField("PIN / passkey", func(v string) { pin = v })
	if s.btPIN != "" {
		field.SetText(s.btPIN)
	}
	var overlay *widgets.Overlay
	finish := func(ok bool) {
		if overlay != nil {
			widget.DismissOverlay(overlay)
		}
		if !ok {
			s.note("pair cancelled")
			return
		}
		s.btPIN = pin
		s.btPair = []string{s.btAddr}
		s.note("pairing " + s.btAddr)
		s.apply()
	}
	cancel := uitoolkit.NewButton("Cancel", func() { finish(false) })
	pair := uitoolkit.NewButton("Pair", func() { finish(true) })
	pair.Primary = true
	field.OnSubmit = func(string) { finish(true) }
	card := uitoolkit.NewPanel("Bluetooth PIN",
		uitoolkit.NewLabel("Enter the PIN or passkey for "+s.btAddr+". Pair stages the device and applies through system-configd."),
		field,
		uitoolkit.NewRow(cancel, pair).WithGap(8),
	)
	card.Raised = true
	overlay = uitoolkit.NewOverlay(card)
	overlay.MinCardH = 180
	if !widget.ShowOverlay(host, overlay) {
		s.btPair = []string{s.btAddr}
		s.note("staged pair " + s.btAddr + " (overlay failed)")
	}
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
	seats := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Seat"}, {Title: "Sessions"},
	}, len(s.seats), func(row, col int) string {
		if row < 0 || row >= len(s.seats) {
			return ""
		}
		x := s.seats[row]
		if col == 1 {
			return fmt.Sprintf("%v", x.Sessions)
		}
		return x.ID
	}, nil)
	inh := "(none)"
	if len(s.inhibits) > 0 {
		inh = fmt.Sprintf("%d idle/sleep inhibit(s)", len(s.inhibits))
		if s.inhibits[0].Who != "" {
			inh += " — " + s.inhibits[0].Who
		}
	}
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Session"),
		uitoolkit.NewLabel("logind sessions + seats. Apply = lock only (loginctl lock-sessions, then coda-hyprlock). Reboot/poweroff are not allowlisted."),
		uitoolkit.NewLabel(fmt.Sprintf("IdleHint %v   inhibit %s", s.idleHint, inh)),
		table, seats,
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
		uitoolkit.NewLabel("Brightness when a backlight exists. Lid HandleLidSwitch via logind drop-in. Suspend/hibernate are gated; reboot/poweroff are not allowlisted."),
		uitoolkit.NewLabel(fmt.Sprintf("CanSuspend %v   CanHibernate %v", s.power.CanSuspend, s.power.CanHibernate)),
		uitoolkit.NewLabel(fmt.Sprintf("Backlight %s  %d / %d", s.power.Backlight, int(s.bright), s.power.MaxBrightness)),
		uitoolkit.NewSlider(0, max, cur, func(v float32) { s.bright = float64(v) }),
		uitoolkit.NewRow(uitoolkit.NewLabel("Lid"), lid).WithGap(8),
		uitoolkit.NewButton("Stage suspend", func() { s.power.Action = "suspend"; s.note("staged suspend") }),
		uitoolkit.NewButton("Stage hibernate", func() { s.power.Action = "hibernate"; s.note("staged hibernate") }),
		s.refreshBtn(protocol.PathPower),
	).WithGap(8)
}

func (s *session) printersPage() uitoolkit.Component {
	note := "CUPS via lpstat / lpadmin. present=false when cups is missing. Apply sets default and enable."
	if len(s.printers.Printers) == 0 {
		note += " No printers (or cups not installed)."
	}
	if s.printers.Default != "" {
		note += " Default: " + s.printers.Default
	}
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Printer"}, {Title: "State"},
	}, len(s.printers.Printers), func(row, col int) string {
		if row < 0 || row >= len(s.printers.Printers) {
			return ""
		}
		p := s.printers.Printers[row]
		if col == 1 {
			return p.State
		}
		if p.Default || p.Name == s.printers.Default {
			return p.Name + " *"
		}
		return p.Name
	}, func(i int) {
		if i >= 0 && i < len(s.printers.Printers) {
			s.printerName = s.printers.Printers[i].Name
			if s.printers.Printers[i].Enabled != nil {
				s.printerOn = *s.printers.Printers[i].Enabled
			} else {
				s.printerOn = true
			}
		}
	})
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Printers"),
		uitoolkit.NewLabel(note),
		table,
		uitoolkit.NewRow(uitoolkit.NewLabel("Default"), uitoolkit.NewTextField(s.printerName, "printer name", func(v string) { s.printerName = v })).WithGap(8),
		uitoolkit.NewSwitch("Enabled", s.printerOn, func(on bool) { s.printerOn = on }),
		s.refreshBtn(protocol.PathPrinters),
	).WithGap(8)
}

func (s *session) usersPage() uitoolkit.Component {
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "User"}, {Title: "UID", Width: 70}, {Title: "Home"}, {Title: "Shell"},
	}, len(s.users.Users), func(row, col int) string {
		if row < 0 || row >= len(s.users.Users) {
			return ""
		}
		u := s.users.Users[row]
		switch col {
		case 1:
			return fmt.Sprintf("%d", u.UID)
		case 2:
			return u.Home
		case 3:
			return u.Shell
		default:
			return u.Name
		}
	}, func(i int) {
		if i >= 0 && i < len(s.users.Users) {
			s.userName = s.users.Users[i].Name
			s.userShell = s.users.Users[i].Shell
		}
	})
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Users"),
		uitoolkit.NewLabel("Local accounts from /etc/passwd. Apply changes login shell only (usermod -s). No add/delete/password."),
		table,
		uitoolkit.NewRow(uitoolkit.NewLabel("User"), uitoolkit.NewTextField(s.userName, "live", func(v string) { s.userName = v })).WithGap(8),
		uitoolkit.NewRow(uitoolkit.NewLabel("Shell"), uitoolkit.NewTextField(s.userShell, "/bin/bash", func(v string) { s.userShell = v })).WithGap(8),
		s.refreshBtn(protocol.PathUsers),
	).WithGap(8)
}

func (s *session) storagePage() uitoolkit.Component {
	note := "lsblk / udisks. present=false when no block devices are reported."
	if len(s.storage.Block) == 0 {
		note += " No disks observed."
	}
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Name"}, {Title: "Type", Width: 70}, {Title: "Size", Width: 80}, {Title: "Mount"}, {Title: "Model"},
	}, len(s.storage.Block), func(row, col int) string {
		if row < 0 || row >= len(s.storage.Block) {
			return ""
		}
		b := s.storage.Block[row]
		switch col {
		case 1:
			return b.Type
		case 2:
			return b.Size
		case 3:
			return b.Mount
		case 4:
			return b.Model
		default:
			return b.Name
		}
	}, func(i int) {
		if i >= 0 && i < len(s.storage.Block) {
			s.storageName = s.storage.Block[i].Name
		}
	})
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Storage"),
		uitoolkit.NewLabel(note+" Apply mount/unmount via udisksctl. System mounts (/, /boot, /usr, /home) are refused."),
		table,
		uitoolkit.NewRow(uitoolkit.NewLabel("Device"), uitoolkit.NewTextField(s.storageName, "sdb1", func(v string) { s.storageName = v })).WithGap(8),
		uitoolkit.NewRow(
			uitoolkit.NewButton("Stage mount", func() { s.storageAct = "mount"; s.note("staged mount " + s.storageName) }),
			uitoolkit.NewButton("Stage unmount", func() { s.storageAct = "unmount"; s.note("staged unmount " + s.storageName) }),
		).WithGap(8),
		s.refreshBtn(protocol.PathStorage),
	).WithGap(8)
}
