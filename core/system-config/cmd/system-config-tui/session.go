package main

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

var pages = []struct {
	Label string
	Path  string
}{
	{"Display", protocol.PathDisplay},
	{"Network", protocol.PathNetwork},
	{"Audio", protocol.PathAudio},
	{"Bluetooth", protocol.PathBluetooth},
	{"Input", protocol.PathInput},
	{"Date & time", protocol.PathDateTime},
	{"Locale", protocol.PathLocale},
	{"Session", protocol.PathSession},
	{"Power", protocol.PathPower},
	{"Printers", protocol.PathPrinters},
	{"Users", protocol.PathUsers},
	{"Storage", protocol.PathStorage},
	{"Devices", protocol.PathDevicesSummary},
	{"PCI", protocol.PathDevicesPCI},
	{"USB", protocol.PathDevicesUSB},
	{"DMI", protocol.PathHardwareDMI},
}

type staged struct {
	scale   float64
	outName string
	outMode string
	outPos  string

	airplane   bool
	wifiDev    string
	wifiSSID   string
	wifiPSK    string
	wifiHidden bool
	wifiDisc   bool
	iface      string
	netMethod  string
	netAddr    string
	netGW      string
	netDNS     string
	netSearch  string

	vol    float64
	mute   bool
	sink   string
	source string

	btPower      bool
	btScan       bool
	btAddr       string
	btPIN        string
	btPair       []string
	btConnect    []string
	btDisconnect []string
	btTrust      []string

	input protocol.InputModel
	dt    protocol.DateTimeModel
	loc   protocol.Locale

	sessionAct string

	bright   float64
	lid      string
	powerAct string

	printerName string
	printerOn   bool
	userName    string
	userShell   string
	storageName string
	storageAct  string
}

type session struct {
	cli *client.Client

	idx    int
	pane   int // 0 sidebar, 1 fields
	field  int
	edit   bool
	buf    string
	status string
	help   bool

	outputs []protocol.Output
	scale   float64
	outName string
	outMode string
	outPos  string
	base    staged

	net        protocol.NetworkModel
	wifiDev    string
	wifiSSID   string
	wifiPSK    string
	wifiHidden bool
	iface      string
	netMethod  string
	netAddr    string
	netGW      string
	netDNS     string
	netSearch  string
	airplane   bool

	audio protocol.AudioModel
	vol   float64
	mute  bool

	bt           protocol.BluetoothModel
	btAddr       string
	btPower      bool
	btScan       bool
	btPIN        string
	btPair       []string
	btConnect    []string
	btDisconnect []string
	btTrust      []string

	input protocol.InputModel
	dt    protocol.DateTimeModel
	loc   protocol.Locale

	summary protocol.DevicesSummary
	pci     []protocol.PCIDevice
	usb     []protocol.USBDevice
	dmi     protocol.DMI

	sessions   []protocol.LoginSession
	seats      []protocol.Seat
	inhibits   []protocol.Inhibit
	idleHint   bool
	sessionAct string
	power      protocol.PowerModel
	bright     float64

	printers    protocol.PrintersModel
	printerName string
	printerOn   bool
	users       protocol.UsersModel
	userName    string
	userShell   string
	storage     protocol.StorageModel
	storageName string
	storageAct  string
}

func newSession(startPath string) (*session, error) {
	c, err := client.Dial(sockpath.Daemon())
	if err != nil {
		return nil, err
	}
	s := &session{cli: c, scale: 1, vol: 0.5}
	s.reload()
	s.snapshotAll()
	if startPath != "" {
		s.goPath(protocol.NormalizePath(startPath))
	}
	s.status = "connected " + sockpath.Daemon()
	return s, nil
}

func (s *session) close() {
	if s.cli != nil {
		_ = s.cli.Close()
	}
}

func (s *session) path() string {
	if s.idx < 0 || s.idx >= len(pages) {
		return protocol.PathDisplay
	}
	return pages[s.idx].Path
}

func (s *session) goPath(path string) {
	path = protocol.NormalizePath(path)
	for i, p := range pages {
		if p.Path == path {
			s.idx = i
			s.field = 0
			return
		}
	}
}

func (s *session) reload() {
	for _, p := range protocol.KnownPaths {
		s.pull(p, true)
	}
	s.snapshotAll()
}

func (s *session) reloadPath(path string) {
	s.pull(path, true)
	if path == protocol.PathDevicesSummary {
		s.pull(protocol.PathDevicesPCI, true)
		s.pull(protocol.PathDevicesUSB, true)
		s.pull(protocol.PathHardwareDMI, true)
	}
	s.snapshot(path)
	s.clearActions(path)
}

func (s *session) pull(path string, refresh bool) {
	if s.cli == nil {
		return
	}
	var resp protocol.Response
	var err error
	if refresh {
		resp, err = s.cli.Refresh(path)
	} else {
		resp, err = s.cli.Get(path)
	}
	if err != nil || !resp.OK {
		return
	}
	s.applyObserved(path, resp.Observed)
}

func (s *session) applyObserved(path string, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	switch path {
	case protocol.PathDisplay:
		var d protocol.DisplayModel
		if json.Unmarshal(raw, &d) != nil {
			return
		}
		s.outputs = d.Outputs
		if len(s.outputs) > 0 {
			o := s.outputs[0]
			for i := range s.outputs {
				if s.outputs[i].Focused {
					o = s.outputs[i]
					break
				}
			}
			s.outName = o.Name
			s.outMode = o.Mode
			s.outPos = o.Position
			if o.Scale > 0 {
				s.scale = o.Scale
			}
		}
	case protocol.PathNetwork:
		_ = json.Unmarshal(raw, &s.net)
		s.wifiDev = s.net.WiFi.Device
		s.wifiSSID = s.net.WiFi.Connected
		s.airplane = s.net.Airplane
		s.wifiPSK = ""
		s.wifiHidden = false
		s.net.WiFi.Disconnect = false
		s.iface = ""
		if len(s.net.Links) > 0 {
			s.iface = s.net.Links[0].Name
		}
		for _, l := range s.net.Links {
			if l.Name == s.iface {
				s.netMethod = l.Method
				if len(l.Addresses) > 0 {
					s.netAddr = l.Addresses[0]
				} else {
					s.netAddr = ""
				}
				s.netGW = l.Gateway
				s.netDNS = strings.Join(l.DNS, " ")
				s.netSearch = strings.Join(l.Search, " ")
				break
			}
		}
	case protocol.PathAudio:
		_ = json.Unmarshal(raw, &s.audio)
		if s.audio.Volume > 0 {
			s.vol = s.audio.Volume
		}
		if s.audio.Mute != nil {
			s.mute = *s.audio.Mute
		}
	case protocol.PathBluetooth:
		_ = json.Unmarshal(raw, &s.bt)
		s.btPower = s.bt.Powered
		s.btScan = s.bt.Scanning
		s.btPair, s.btConnect, s.btDisconnect, s.btTrust = nil, nil, nil, nil
		s.btPIN = ""
	case protocol.PathInput:
		_ = json.Unmarshal(raw, &s.input)
	case protocol.PathDateTime:
		_ = json.Unmarshal(raw, &s.dt)
	case protocol.PathLocale:
		_ = json.Unmarshal(raw, &s.loc)
	case protocol.PathDevicesSummary:
		_ = json.Unmarshal(raw, &s.summary)
	case protocol.PathDevicesPCI:
		var list protocol.PCIList
		if json.Unmarshal(raw, &list) == nil {
			s.pci = list.Devices
		}
	case protocol.PathDevicesUSB:
		var list protocol.USBList
		if json.Unmarshal(raw, &list) == nil {
			s.usb = list.Devices
		}
	case protocol.PathHardwareDMI:
		_ = json.Unmarshal(raw, &s.dmi)
	case protocol.PathSession:
		var sm protocol.SessionModel
		if json.Unmarshal(raw, &sm) == nil {
			s.sessions = sm.Sessions
			s.seats = sm.Seats
			s.inhibits = sm.IdleInhibit
			s.idleHint = sm.IdleHint
		}
		s.sessionAct = ""
	case protocol.PathPrinters:
		_ = json.Unmarshal(raw, &s.printers)
	case protocol.PathUsers:
		_ = json.Unmarshal(raw, &s.users)
	case protocol.PathStorage:
		_ = json.Unmarshal(raw, &s.storage)
		s.storageAct = ""
	case protocol.PathPower:
		_ = json.Unmarshal(raw, &s.power)
		s.bright = float64(s.power.Brightness)
		s.power.Action = ""
	}
}

func (s *session) capture() staged {
	return staged{
		scale: s.scale, outName: s.outName, outMode: s.outMode, outPos: s.outPos,
		airplane: s.airplane, wifiDev: s.wifiDev, wifiSSID: s.wifiSSID, wifiPSK: s.wifiPSK,
		wifiHidden: s.wifiHidden, wifiDisc: s.net.WiFi.Disconnect,
		iface: s.iface, netMethod: s.netMethod, netAddr: s.netAddr, netGW: s.netGW,
		netDNS: s.netDNS, netSearch: s.netSearch,
		vol: s.vol, mute: s.mute, sink: s.audio.DefaultSink, source: s.audio.DefaultSource,
		btPower: s.btPower, btScan: s.btScan, btAddr: s.btAddr, btPIN: s.btPIN,
		btPair: slices.Clone(s.btPair), btConnect: slices.Clone(s.btConnect),
		btDisconnect: slices.Clone(s.btDisconnect), btTrust: slices.Clone(s.btTrust),
		input: s.input, dt: s.dt, loc: s.loc,
		sessionAct: s.sessionAct,
		bright:     s.bright, lid: s.power.Lid, powerAct: s.power.Action,
		printerName: s.printerName, printerOn: s.printerOn,
		userName: s.userName, userShell: s.userShell,
		storageName: s.storageName, storageAct: s.storageAct,
	}
}

func copyPath(dst *staged, src staged, path string) {
	switch path {
	case protocol.PathDisplay:
		dst.scale, dst.outName, dst.outMode, dst.outPos = src.scale, src.outName, src.outMode, src.outPos
	case protocol.PathNetwork:
		dst.airplane, dst.wifiDev, dst.wifiSSID, dst.wifiPSK = src.airplane, src.wifiDev, src.wifiSSID, src.wifiPSK
		dst.wifiHidden, dst.wifiDisc = src.wifiHidden, src.wifiDisc
		dst.iface, dst.netMethod, dst.netAddr, dst.netGW = src.iface, src.netMethod, src.netAddr, src.netGW
		dst.netDNS, dst.netSearch = src.netDNS, src.netSearch
	case protocol.PathAudio:
		dst.vol, dst.mute, dst.sink, dst.source = src.vol, src.mute, src.sink, src.source
	case protocol.PathBluetooth:
		dst.btPower, dst.btScan, dst.btAddr, dst.btPIN = src.btPower, src.btScan, src.btAddr, src.btPIN
		dst.btPair = slices.Clone(src.btPair)
		dst.btConnect = slices.Clone(src.btConnect)
		dst.btDisconnect = slices.Clone(src.btDisconnect)
		dst.btTrust = slices.Clone(src.btTrust)
	case protocol.PathInput:
		dst.input = src.input
	case protocol.PathDateTime:
		dst.dt = src.dt
	case protocol.PathLocale:
		dst.loc = src.loc
	case protocol.PathSession:
		dst.sessionAct = src.sessionAct
	case protocol.PathPower:
		dst.bright, dst.lid, dst.powerAct = src.bright, src.lid, src.powerAct
	case protocol.PathPrinters:
		dst.printerName, dst.printerOn = src.printerName, src.printerOn
	case protocol.PathUsers:
		dst.userName, dst.userShell = src.userName, src.userShell
	case protocol.PathStorage:
		dst.storageName, dst.storageAct = src.storageName, src.storageAct
	}
}

func (s *session) snapshot(path string) { copyPath(&s.base, s.capture(), path) }
func (s *session) snapshotAll()         { s.base = s.capture() }

func near(a, b float64) bool { return abs(a-b) <= 0.01 }

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (s *session) dirtyPath(path string) bool {
	if s.cli == nil || !protocol.Settable(path) {
		return false
	}
	cur, base := s.capture(), s.base
	switch path {
	case protocol.PathDisplay:
		return !near(cur.scale, base.scale) || cur.outName != base.outName ||
			cur.outMode != base.outMode || cur.outPos != base.outPos
	case protocol.PathNetwork:
		return cur.airplane != base.airplane || cur.wifiDev != base.wifiDev ||
			cur.wifiSSID != base.wifiSSID || cur.wifiPSK != base.wifiPSK ||
			cur.wifiHidden != base.wifiHidden || cur.wifiDisc != base.wifiDisc ||
			cur.iface != base.iface || cur.netMethod != base.netMethod ||
			cur.netAddr != base.netAddr || cur.netGW != base.netGW ||
			cur.netDNS != base.netDNS || cur.netSearch != base.netSearch
	case protocol.PathAudio:
		return !near(cur.vol, base.vol) || cur.mute != base.mute ||
			cur.sink != base.sink || cur.source != base.source
	case protocol.PathBluetooth:
		return cur.btPower != base.btPower || cur.btScan != base.btScan ||
			cur.btPIN != base.btPIN ||
			!slices.Equal(cur.btPair, base.btPair) ||
			!slices.Equal(cur.btConnect, base.btConnect) ||
			!slices.Equal(cur.btDisconnect, base.btDisconnect) ||
			!slices.Equal(cur.btTrust, base.btTrust)
	case protocol.PathInput:
		return cur.input.KBLayout != base.input.KBLayout ||
			cur.input.Keymap != base.input.Keymap ||
			!near(cur.input.PointerSpeed, base.input.PointerSpeed) ||
			cur.input.NaturalScroll != base.input.NaturalScroll ||
			cur.input.TapToClick != base.input.TapToClick
	case protocol.PathDateTime:
		return cur.dt.Timezone != base.dt.Timezone || cur.dt.NTP != base.dt.NTP ||
			cur.dt.Time != base.dt.Time
	case protocol.PathLocale:
		return cur.loc.Lang != base.loc.Lang || cur.loc.Keymap != base.loc.Keymap ||
			cur.loc.Timezone != base.loc.Timezone
	case protocol.PathSession:
		return strings.TrimSpace(cur.sessionAct) != "" && cur.sessionAct != base.sessionAct
	case protocol.PathPower:
		return !near(cur.bright, base.bright) || cur.lid != base.lid ||
			cur.powerAct != base.powerAct
	case protocol.PathPrinters:
		return cur.printerName != base.printerName || cur.printerOn != base.printerOn
	case protocol.PathUsers:
		return cur.userName != base.userName || cur.userShell != base.userShell
	case protocol.PathStorage:
		return cur.storageAct != "" && (cur.storageAct != base.storageAct ||
			cur.storageName != base.storageName)
	default:
		return false
	}
}

func (s *session) clearActions(path string) {
	switch path {
	case protocol.PathNetwork:
		s.net.WiFi.Disconnect = false
	case protocol.PathBluetooth:
		s.btPair, s.btConnect, s.btDisconnect, s.btTrust = nil, nil, nil, nil
		s.btPIN = ""
	case protocol.PathSession:
		s.sessionAct = ""
	case protocol.PathPower:
		s.power.Action = ""
	case protocol.PathStorage:
		s.storageAct = ""
	}
}

func (s *session) desiredJSON(path string) (json.RawMessage, error) {
	var v any
	switch path {
	case protocol.PathDisplay:
		v = protocol.DisplayModel{Outputs: []protocol.Output{{
			Name: s.outName, Scale: s.scale, Mode: s.outMode, Position: s.outPos,
		}}}
	case protocol.PathNetwork:
		n := protocol.NetworkModel{
			Airplane: s.airplane,
			WiFi: protocol.WiFiState{
				Device: s.wifiDev, Connect: s.wifiSSID, PSK: s.wifiPSK,
				Hidden: s.wifiHidden, Disconnect: s.net.WiFi.Disconnect,
			},
		}
		if s.iface != "" {
			link := protocol.NetLink{Name: s.iface, Enabled: true, Method: s.netMethod, Gateway: s.netGW}
			if s.netAddr != "" {
				link.Addresses = []string{s.netAddr}
			}
			if s.netDNS != "" {
				link.DNS = strings.Fields(s.netDNS)
			}
			if s.netSearch != "" {
				link.Search = strings.Fields(s.netSearch)
			}
			n.Links = []protocol.NetLink{link}
		}
		v = n
	case protocol.PathAudio:
		m := s.mute
		v = protocol.AudioModel{
			DefaultSink: s.audio.DefaultSink, DefaultSource: s.audio.DefaultSource,
			Volume: s.vol, Mute: &m, Sinks: s.audio.Sinks, Sources: s.audio.Sources,
		}
	case protocol.PathBluetooth:
		v = protocol.BluetoothModel{
			Powered:    s.btPower,
			Scanning:   s.btScan,
			Pair:       s.btPair,
			Connect:    s.btConnect,
			Disconnect: s.btDisconnect,
			Trust:      s.btTrust,
			PIN:        s.btPIN,
		}
	case protocol.PathInput:
		v = s.input
	case protocol.PathDateTime:
		v = s.dt
	case protocol.PathLocale:
		v = s.loc
	case protocol.PathSession:
		if strings.TrimSpace(s.sessionAct) == "" {
			return nil, fmt.Errorf("nothing to apply")
		}
		v = protocol.SessionModel{Action: s.sessionAct}
	case protocol.PathPower:
		v = protocol.PowerModel{Brightness: int(s.bright), Backlight: s.power.Backlight, Action: s.power.Action, Lid: s.power.Lid}
	case protocol.PathPrinters:
		en := s.printerOn
		p := protocol.PrintersModel{Default: s.printerName}
		if s.printerName != "" {
			p.Printers = []protocol.Printer{{Name: s.printerName, Enabled: &en, Default: true}}
		}
		v = p
	case protocol.PathUsers:
		v = protocol.UsersModel{Users: []protocol.LocalUser{{Name: s.userName, Shell: s.userShell}}}
	case protocol.PathStorage:
		v = protocol.StorageModel{Block: []protocol.BlockDev{{Name: s.storageName, Action: s.storageAct}}}
	default:
		return nil, fmt.Errorf("observe-only")
	}
	return json.Marshal(v)
}

func (s *session) applyPath(path string) error {
	if s.cli == nil {
		return fmt.Errorf("system-configd not running")
	}
	if !protocol.Settable(path) {
		return fmt.Errorf("observe-only")
	}
	if !s.dirtyPath(path) {
		return fmt.Errorf("nothing to apply")
	}
	data, err := s.desiredJSON(path)
	if err != nil {
		return err
	}
	if _, err := s.cli.Set(path, data); err != nil {
		return err
	}
	resp, err := s.cli.Apply(path)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	s.clearActions(path)
	s.snapshot(path)
	return nil
}

func (s *session) dump(w io.Writer) error {
	fmt.Fprintln(w, "Coda system-config TUI — D only")
	fmt.Fprintln(w, "Pages: refresh / edit / per-section Apply (settable paths). Observe-only have no Apply.")
	for _, p := range protocol.KnownPaths {
		s.pull(p, false)
		kind := "observe"
		if protocol.Settable(p) {
			kind = "settable"
		}
		st := ""
		if s.cli != nil {
			if resp, err := s.cli.Get(p); err == nil && resp.Status != nil {
				st = fmt.Sprintf(" present=%v configured=%v changed=%v", resp.Status.Present, resp.Status.Configured, resp.Status.Changed)
			}
		}
		fmt.Fprintf(w, "  %s  (%s)%s\n", p, kind, st)
	}
	return nil
}

func pagePaths() []string {
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = p.Path
	}
	return out
}
