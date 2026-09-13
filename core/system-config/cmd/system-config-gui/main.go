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

	page       int // 0 display, 1 devices
	outputs    []protocol.Output
	pci        []protocol.PCIDevice
	pciTotal   int
	summary    protocol.DevicesSummary
	scale      float64 // staged
	savedScale float64
	outName    string
	status     *widgets.StatusBar
	applyBtn   *widgets.Button
	scaleLbl   *widgets.Label
}

func newSession(a *app.Application, win *app.Window, cli *client.Client) *session {
	s := &session{app: a, win: win, cli: cli, scale: 1, savedScale: 1}
	s.reload()
	return s
}

func (s *session) reload() {
	if s.cli == nil {
		return
	}
	if resp, err := s.cli.Refresh(protocol.PathDisplay); err == nil && resp.OK && len(resp.Observed) > 0 {
		var d protocol.DisplayModel
		if json.Unmarshal(resp.Observed, &d) == nil {
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
					s.scale = o.Scale
					s.savedScale = o.Scale
				}
			}
		}
	}
	if resp, err := s.cli.Refresh(protocol.PathDevicesSummary); err == nil && resp.OK {
		_ = json.Unmarshal(resp.Observed, &s.summary)
	}
	if resp, err := s.cli.Refresh(protocol.PathDevicesPCI); err == nil && resp.OK {
		var list protocol.PCIList
		if json.Unmarshal(resp.Observed, &list) == nil {
			s.pciTotal = len(list.Devices)
			s.pci = list.Devices
			if len(s.pci) > pciCap {
				s.pci = s.pci[:pciCap]
			}
		}
	}
}

func (s *session) dirty() bool {
	return s.scale > 0 && abs(s.scale-s.savedScale) > 0.01
}

func (s *session) note(msg string) {
	if s.status != nil {
		s.status.Set(0, msg)
	}
}

func (s *session) rebuild() {
	s.win.SetContent(s.build())
}

func (s *session) build() uitoolkit.Component {
	disp := uitoolkit.NewTreeNode("Display")
	devs := uitoolkit.NewTreeNode("Devices")
	tree := uitoolkit.NewTreeView(disp, devs)
	switch s.page {
	case 1:
		tree.Selected = devs
	default:
		tree.Selected = disp
	}
	tree.OnSelect = func(n *widgets.TreeNode) {
		if n == nil {
			return
		}
		next := 0
		if n.Label == "Devices" {
			next = 1
		}
		if next != s.page {
			s.page = next
			s.rebuild()
		}
	}

	side := uitoolkit.NewColumn(
		uitoolkit.NewTitle("Settings"),
		uitoolkit.NewLabel("system-config"),
		tree,
	)
	side.WithGap(8).WithPad(10)
	side.AddFlex(tree, 1)

	var page uitoolkit.Component
	if s.page == 1 {
		page = s.devicesPage()
	} else {
		page = s.displayPage()
	}
	split := uitoolkit.NewSplitter(true, side, uitoolkit.NewPad(12, page))
	split.Ratio = 0.24

	s.applyBtn = uitoolkit.NewButton("Apply", s.apply)
	s.applyBtn.Primary = true
	s.applyBtn.SetEnabled(s.dirty() && s.cli != nil)
	hint := uitoolkit.NewLabel("Apply sends staged display scale to system-configd. Close without Apply discards it.")
	actions := uitoolkit.NewRow(s.applyBtn, hint).WithGap(12).WithPad(8)

	st := "system-configd connected"
	if s.cli == nil {
		st = "system-configd not running — CLI/GUI still paint; Apply is disabled"
	}
	s.status = uitoolkit.NewStatusBar(st, sockpath.Daemon(), "v1")

	chrome := uitoolkit.NewTitleBar("Coda Settings", "Display and devices via system-configd (uitoolkit)")
	root := uitoolkit.NewColumn(chrome, split, actions, s.status)
	root.AddFlex(split, 1)
	return root
}

func (s *session) displayPage() uitoolkit.Component {
	rows := len(s.outputs)
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Output"},
		{Title: "Mode", Width: 140},
		{Title: "Scale", Width: 80},
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
			name := o.Name
			if o.Name == s.outName {
				name += "  (focus)"
			}
			return name
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
	slider := uitoolkit.NewSlider(100, 200, pct, func(v float32) {
		stage(float64(v) / 100)
	})
	num := uitoolkit.NewNumberField(1, 2, s.scale, 0.25, stage)

	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Display"),
		uitoolkit.NewLabel("Stage a scale, then Apply. D runs hyprctl eval hl.monitor (not keyword)."),
		table,
		s.scaleLbl,
		slider,
		uitoolkit.NewRow(uitoolkit.NewLabel("Factor"), num).WithGap(8),
		uitoolkit.NewButton("Refresh", func() {
			s.reload()
			s.rebuild()
			s.note("refreshed display")
		}),
	).WithGap(8)
}

func (s *session) devicesPage() uitoolkit.Component {
	summary := fmt.Sprintf("%s %s   PCI %d   USB %d", s.summary.Vendor, s.summary.Product, s.summary.PCICount, s.summary.USBCount)
	if s.pciTotal > pciCap {
		summary += fmt.Sprintf("   (table shows %d of %d — TableView is alpha)", pciCap, s.pciTotal)
	}
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "ID"},
		{Title: "Vendor", Width: 90},
		{Title: "Device", Width: 90},
		{Title: "Class", Width: 90},
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
	table.Mono = true
	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Devices"),
		uitoolkit.NewLabel(summary),
		table,
		uitoolkit.NewButton("Refresh", func() {
			s.reload()
			s.rebuild()
			s.note("refreshed devices")
		}),
	).WithGap(8)
}

func (s *session) apply() {
	if s.cli == nil {
		s.note("system-configd not running")
		return
	}
	if s.outName == "" {
		s.note("no output")
		return
	}
	data, _ := json.Marshal(protocol.DisplayModel{
		Outputs: []protocol.Output{{Name: s.outName, Scale: s.scale}},
	})
	if _, err := s.cli.Set(protocol.PathDisplay, data); err != nil {
		s.note(err.Error())
		return
	}
	resp, err := s.cli.Apply(protocol.PathDisplay)
	if err != nil {
		s.note(err.Error())
		return
	}
	if !resp.OK {
		s.note(resp.Error)
		return
	}
	s.savedScale = s.scale
	if s.applyBtn != nil {
		s.applyBtn.SetEnabled(false)
	}
	s.note(fmt.Sprintf("applied scale %g on %s", s.scale, s.outName))
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
