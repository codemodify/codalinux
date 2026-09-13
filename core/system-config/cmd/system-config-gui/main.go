// Command system-config-gui is the Settings client (uitoolkit Mail/Settings pattern).
// App-level Unix socket + JSON-lines RPC. Talks to system-configd only.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/widgets"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

const pciCap = 256

var nav = []struct {
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
	{"Devices", protocol.PathDevicesSummary},
	{"Session", protocol.PathSession},
	{"Power", protocol.PathPower},
}

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

	outputs    []protocol.Output
	scale      float64
	savedScale float64
	outName    string

	net      protocol.NetworkModel
	wifiDev  string
	wifiSSID string
	wifiPSK  string
	iface    string

	audio protocol.AudioModel
	vol   float64
	mute  bool

	bt      protocol.BluetoothModel
	btAddr  string
	btPower bool

	input protocol.InputModel
	dt    protocol.DateTimeModel
	loc   protocol.Locale

	summary  protocol.DevicesSummary
	pci      []protocol.PCIDevice
	pciTotal int
	usb      []protocol.USBDevice
	dmi      protocol.DMI

	sessions []protocol.LoginSession
	power    protocol.PowerModel
	bright   float64

	status   *widgets.StatusBar
	applyBtn *widgets.Button
	scaleLbl *widgets.Label
	volLbl   *widgets.Label
}

func newSession(a *app.Application, win *app.Window, cli *client.Client) *session {
	s := &session{app: a, win: win, cli: cli, scale: 1, savedScale: 1, vol: 0.5}
	s.reload()
	return s
}

func (s *session) path() string { return nav[s.page].Path }

func (s *session) reload() {
	if s.cli == nil {
		return
	}
	refresh := func(path string) json.RawMessage {
		resp, err := s.cli.Refresh(path)
		if err != nil || !resp.OK {
			return nil
		}
		return resp.Observed
	}
	if raw := refresh(protocol.PathDisplay); len(raw) > 0 {
		var d protocol.DisplayModel
		if json.Unmarshal(raw, &d) == nil {
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
				if o.Scale > 0 {
					s.scale, s.savedScale = o.Scale, o.Scale
				}
			}
		}
	}
	if raw := refresh(protocol.PathNetwork); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.net)
		s.wifiDev = s.net.WiFi.Device
		s.wifiSSID = s.net.WiFi.Connected
		if s.iface == "" && len(s.net.Links) > 0 {
			s.iface = s.net.Links[0].Name
		}
	}
	if raw := refresh(protocol.PathAudio); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.audio)
		if s.audio.Volume > 0 {
			s.vol = s.audio.Volume
		}
		if s.audio.Mute != nil {
			s.mute = *s.audio.Mute
		}
	}
	if raw := refresh(protocol.PathBluetooth); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.bt)
		s.btPower = s.bt.Powered
	}
	if raw := refresh(protocol.PathInput); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.input)
	}
	if raw := refresh(protocol.PathDateTime); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.dt)
	}
	if raw := refresh(protocol.PathLocale); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.loc)
	}
	if raw := refresh(protocol.PathDevicesSummary); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.summary)
	}
	if raw := refresh(protocol.PathDevicesPCI); len(raw) > 0 {
		var list protocol.PCIList
		if json.Unmarshal(raw, &list) == nil {
			s.pciTotal = len(list.Devices)
			s.pci = list.Devices
			if len(s.pci) > pciCap {
				s.pci = s.pci[:pciCap]
			}
		}
	}
	if raw := refresh(protocol.PathDevicesUSB); len(raw) > 0 {
		var list protocol.USBList
		if json.Unmarshal(raw, &list) == nil {
			s.usb = list.Devices
		}
	}
	if raw := refresh(protocol.PathHardwareDMI); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.dmi)
	}
	if raw := refresh(protocol.PathSession); len(raw) > 0 {
		var sm protocol.SessionModel
		if json.Unmarshal(raw, &sm) == nil {
			s.sessions = sm.Sessions
		}
	}
	if raw := refresh(protocol.PathPower); len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.power)
		s.bright = float64(s.power.Brightness)
	}
}

func (s *session) dirty() bool {
	if s.cli == nil {
		return false
	}
	switch s.path() {
	case protocol.PathDisplay:
		return s.scale > 0 && abs(s.scale-s.savedScale) > 0.01
	case protocol.PathDevicesSummary:
		return false
	default:
		return true
	}
}

func (s *session) note(msg string) {
	if s.status != nil {
		s.status.Set(0, msg)
	}
}

func (s *session) rebuild() { s.win.SetContent(s.build()) }

func (s *session) build() uitoolkit.Component {
	nodes := make([]*widgets.TreeNode, len(nav))
	for i, n := range nav {
		nodes[i] = uitoolkit.NewTreeNode(n.Label)
	}
	tree := uitoolkit.NewTreeView(nodes...)
	if s.page >= 0 && s.page < len(nodes) {
		tree.Selected = nodes[s.page]
	}
	tree.OnSelect = func(n *widgets.TreeNode) {
		if n == nil {
			return
		}
		for i, item := range nav {
			if n.Label == item.Label && i != s.page {
				s.page = i
				s.rebuild()
				return
			}
		}
	}

	side := uitoolkit.NewColumn(
		uitoolkit.NewTitle("Settings"),
		uitoolkit.NewLabel("system-config"),
		tree,
	)
	side.WithGap(8).WithPad(10)
	side.AddFlex(tree, 1)

	page := s.pageFor(s.path())
	split := uitoolkit.NewSplitter(true, side, uitoolkit.NewPad(12, page))
	split.Ratio = 0.22

	s.applyBtn = uitoolkit.NewButton("Apply", s.apply)
	s.applyBtn.Primary = true
	s.applyBtn.SetEnabled(s.dirty() && s.cli != nil && protocol.Settable(s.path()))
	hint := uitoolkit.NewLabel("Apply sends staged desired state to system-configd. Observe-only pages have no Apply.")
	actions := uitoolkit.NewRow(s.applyBtn, hint).WithGap(12).WithPad(8)

	st := "system-configd connected"
	if s.cli == nil {
		st = "system-configd not running — CLI/GUI still paint; Apply is disabled"
	}
	s.status = uitoolkit.NewStatusBar(st, sockpath.Daemon(), "v1")

	chrome := uitoolkit.NewTitleBar("Coda Settings", "All domains via system-configd (uitoolkit)")
	root := uitoolkit.NewColumn(chrome, split, actions, s.status)
	root.AddFlex(split, 1)
	return root
}

func (s *session) apply() {
	if s.cli == nil {
		s.note("system-configd not running")
		return
	}
	path := s.path()
	if !protocol.Settable(path) {
		s.note("observe-only")
		return
	}
	data, err := s.desiredJSON()
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
	if path == protocol.PathDisplay {
		s.savedScale = s.scale
	}
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(false)
	}
	s.note("applied " + path)
}

func (s *session) desiredJSON() (json.RawMessage, error) {
	var v any
	switch s.path() {
	case protocol.PathDisplay:
		v = protocol.DisplayModel{Outputs: []protocol.Output{{Name: s.outName, Scale: s.scale}}}
	case protocol.PathNetwork:
		n := protocol.NetworkModel{WiFi: protocol.WiFiState{Device: s.wifiDev, Connect: s.wifiSSID, PSK: s.wifiPSK}}
		if s.iface != "" {
			n.Links = []protocol.NetLink{{Name: s.iface, Enabled: true}}
		}
		v = n
	case protocol.PathAudio:
		m := s.mute
		v = protocol.AudioModel{DefaultSink: s.audio.DefaultSink, Volume: s.vol, Mute: &m}
	case protocol.PathBluetooth:
		bt := protocol.BluetoothModel{Powered: s.btPower}
		if s.btAddr != "" {
			bt.Connect = []string{s.btAddr}
		}
		v = bt
	case protocol.PathInput:
		v = s.input
	case protocol.PathDateTime:
		v = s.dt
	case protocol.PathLocale:
		v = s.loc
	case protocol.PathSession:
		v = protocol.SessionModel{Action: "lock"}
	case protocol.PathPower:
		v = protocol.PowerModel{Brightness: int(s.bright), Backlight: s.power.Backlight, Action: s.power.Action, Lid: s.power.Lid}
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
