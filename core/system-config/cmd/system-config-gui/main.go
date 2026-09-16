// Command system-config-gui is the Settings client (composed prefs shell).
// App-level Unix socket + JSON-lines RPC. Talks to system-configd only.
// uitoolkit has no PrefsPage / NavRail; the shell is ListView + section chrome.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/widgets"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

const pciCap = 256

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write system-config-gui.png")
	out := flag.String("screenshot", "", "write PNG and exit")
	flag.Parse()

	c, err := client.Dial(sockpath.Daemon())
	if err != nil {
		log.Printf("system-configd not running: %v", err)
		c = nil
	} else {
		defer c.Close()
	}

	off := *headless || *out != ""
	a := uitoolkit.New(uitoolkit.Options{Headless: off})
	win, err := a.NewWindow(uitoolkit.WindowOptions{
		Title: "Coda Settings", Width: 1024, Height: 720, MinWidth: 720, MinHeight: 520,
		Headless: off,
	})
	if err != nil {
		log.Fatal(err)
	}
	s := newSession(a, win, c)
	win.SetContent(s.build())
	if off {
		path := "system-config-gui.png"
		if *out != "" {
			path = *out
		}
		if err := win.WritePNG(path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", path)
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

type session struct {
	app *app.Application
	win *app.Window
	cli *client.Client

	page int

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
	btPair       []string
	btConnect    []string
	btDisconnect []string
	btTrust      []string
	btPIN        string

	input protocol.InputModel
	dt    protocol.DateTimeModel
	loc   protocol.Locale

	summary  protocol.DevicesSummary
	pci      []protocol.PCIDevice
	pciTotal int
	usb      []protocol.USBDevice
	dmi      protocol.DMI

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

	status   *widgets.StatusBar
	applyBtn *widgets.Button
	navList  *widgets.ListView
	scaleLbl *widgets.Label
	volLbl   *widgets.Label
}

func newSession(a *app.Application, win *app.Window, cli *client.Client) *session {
	s := &session{app: a, win: win, cli: cli, scale: 1, vol: 0.5}
	s.reload()
	s.snapshotAll()
	return s
}

func (s *session) path() string { return nav[s.page].Path }

func (s *session) refreshRaw(path string) json.RawMessage {
	if s.cli == nil {
		return nil
	}
	resp, err := s.cli.Refresh(path)
	if err != nil || !resp.OK {
		return nil
	}
	return resp.Observed
}

func (s *session) reload() {
	for _, p := range protocol.KnownPaths {
		s.pull(p)
	}
	s.snapshotAll()
}

func (s *session) reloadPath(path string) {
	s.pull(path)
	if path == protocol.PathDevicesSummary {
		s.pull(protocol.PathDevicesPCI)
		s.pull(protocol.PathDevicesUSB)
		s.pull(protocol.PathHardwareDMI)
	}
	s.snapshot(path)
}

func (s *session) pull(path string) {
	raw := s.refreshRaw(path)
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
		return
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
		return
	case protocol.PathAudio:
		_ = json.Unmarshal(raw, &s.audio)
		if s.audio.Volume > 0 {
			s.vol = s.audio.Volume
		}
		if s.audio.Mute != nil {
			s.mute = *s.audio.Mute
		}
		return
	case protocol.PathBluetooth:
		_ = json.Unmarshal(raw, &s.bt)
		s.btPower = s.bt.Powered
		s.btScan = s.bt.Scanning
		s.btPair, s.btConnect, s.btDisconnect, s.btTrust = nil, nil, nil, nil
		s.btPIN = ""
		return
	case protocol.PathInput:
		_ = json.Unmarshal(raw, &s.input)
		return
	case protocol.PathDateTime:
		_ = json.Unmarshal(raw, &s.dt)
		return
	case protocol.PathLocale:
		_ = json.Unmarshal(raw, &s.loc)
		return
	case protocol.PathDevicesSummary:
		_ = json.Unmarshal(raw, &s.summary)
		return
	case protocol.PathDevicesPCI:
		var list protocol.PCIList
		if json.Unmarshal(raw, &list) == nil {
			s.pciTotal = len(list.Devices)
			s.pci = list.Devices
			if len(s.pci) > pciCap {
				s.pci = s.pci[:pciCap]
			}
		}
		return
	case protocol.PathDevicesUSB:
		var list protocol.USBList
		if json.Unmarshal(raw, &list) == nil {
			s.usb = list.Devices
		}
		return
	case protocol.PathHardwareDMI:
		_ = json.Unmarshal(raw, &s.dmi)
		return
	case protocol.PathSession:
		var sm protocol.SessionModel
		if json.Unmarshal(raw, &sm) == nil {
			s.sessions = sm.Sessions
			s.seats = sm.Seats
			s.inhibits = sm.IdleInhibit
			s.idleHint = sm.IdleHint
		}
		s.sessionAct = ""
		return
	case protocol.PathPrinters:
		_ = json.Unmarshal(raw, &s.printers)
		return
	case protocol.PathUsers:
		_ = json.Unmarshal(raw, &s.users)
		return
	case protocol.PathStorage:
		_ = json.Unmarshal(raw, &s.storage)
		s.storageAct = ""
		return
	case protocol.PathPower:
		_ = json.Unmarshal(raw, &s.power)
		s.bright = float64(s.power.Brightness)
		s.power.Action = ""
		return
	}
}

func (s *session) note(msg string) {
	if s.status != nil {
		s.status.Set(0, msg)
	}
}

func (s *session) rebuild() { s.win.SetContent(s.build()) }

func (s *session) apply() { s.applyPath(s.path()) }

func (s *session) applyPath(path string) {
	if s.cli == nil {
		s.note("system-configd not running")
		return
	}
	if !protocol.Settable(path) {
		s.note("observe-only")
		return
	}
	if !s.dirtyPath(path) {
		s.note("nothing to apply")
		return
	}
	data, err := s.desiredJSON(path)
	if err != nil {
		s.note(err.Error())
		return
	}
	if _, err := s.cli.Set(path, data); err != nil {
		s.note(err.Error())
		return
	}
	resp, err := s.cli.Apply(path)
	if err != nil {
		s.note(err.Error())
		return
	}
	if !resp.OK {
		s.note(resp.Error)
		return
	}
	s.clearActions(path)
	s.snapshot(path)
	s.syncApply()
	s.note("applied " + path)
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
		return nil, fmt.Errorf("nothing to apply")
	}
	return json.Marshal(v)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
