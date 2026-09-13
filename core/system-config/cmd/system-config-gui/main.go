// Command system-config-gui is the Settings client built with uitoolkit.
// Talks to system-configd only.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/sockpath"
)

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
		Title: "Coda Settings", Width: 960, Height: 640, MinWidth: 720, MinHeight: 480,
		Headless: off,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(build(a, win, c, 0, ""))
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

func build(a *app.Application, win *app.Window, c *client.Client, page int, note string) uitoolkit.Component {
	pages := []string{"Display", "Devices"}
	if page < 0 || page >= len(pages) {
		page = 0
	}
	nav := uitoolkit.NewListView(len(pages), func(i int) string { return pages[i] }, func(i int) {
		win.SetContent(build(a, win, c, i, note))
	})
	nav.Selected = page

	var body uitoolkit.Component
	switch page {
	case 1:
		body = devicesPage(a, win, c, page)
	default:
		body = displayPage(a, win, c, page)
	}

	st := "system-configd connected"
	if c == nil {
		st = "system-configd not running"
	}
	if note != "" {
		st = note
	}
	return uitoolkit.NewColumn(
		uitoolkit.NewTitleBar("Coda Settings", "Display and devices via system-configd"),
		uitoolkit.NewSplitter(false, uitoolkit.NewPad(8, nav), uitoolkit.NewPad(12, body)),
		uitoolkit.NewStatusBar(st, sockpath.Daemon(), "v1"),
	)
}

func displayPage(a *app.Application, win *app.Window, c *client.Client, page int) uitoolkit.Component {
	var disp protocol.DisplayModel
	if c != nil {
		if resp, err := c.Refresh("display"); err == nil && resp.OK && len(resp.Observed) > 0 {
			_ = json.Unmarshal(resp.Observed, &disp)
		} else if resp, err := c.Get("display"); err == nil && len(resp.Observed) > 0 {
			_ = json.Unmarshal(resp.Observed, &disp)
		}
	}
	rows := len(disp.Outputs)
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "Output"},
		{Title: "Mode", Width: 140},
		{Title: "Scale", Width: 70},
	}, rows, func(row, col int) string {
		if row < 0 || row >= len(disp.Outputs) {
			return ""
		}
		o := disp.Outputs[row]
		switch col {
		case 1:
			return o.Mode
		case 2:
			return fmt.Sprintf("%g", o.Scale)
		default:
			return o.Name
		}
	}, nil)

	setScale := func(scale float64) {
		if c == nil || len(disp.Outputs) == 0 {
			win.SetContent(build(a, win, c, page, "no display / daemon"))
			return
		}
		name := disp.Outputs[0].Name
		for _, o := range disp.Outputs {
			if o.Focused {
				name = o.Name
				break
			}
		}
		data, _ := json.Marshal(protocol.DisplayModel{Outputs: []protocol.Output{{Name: name, Scale: scale}}})
		if _, err := c.Set("display", data); err != nil {
			win.SetContent(build(a, win, c, page, err.Error()))
			return
		}
		resp, err := c.Apply("display")
		note := "applied scale"
		if err != nil {
			note = err.Error()
		} else if !resp.OK {
			note = resp.Error
		}
		win.SetContent(build(a, win, c, page, note))
	}

	return uitoolkit.NewColumn(
		uitoolkit.NewTitle("Display"),
		uitoolkit.NewLabel("Scale goes to system-configd, then apply (hyprctl eval hl.monitor)."),
		table,
		uitoolkit.NewRow(
			uitoolkit.NewButton("100%", func() { setScale(1) }),
			uitoolkit.NewButton("125%", func() { setScale(1.25) }),
			uitoolkit.NewButton("200%", func() { setScale(2) }),
			uitoolkit.NewButton("Refresh", func() { win.SetContent(build(a, win, c, page, "refreshed")) }),
		),
	)
}

func devicesPage(a *app.Application, win *app.Window, c *client.Client, page int) uitoolkit.Component {
	var sum protocol.DevicesSummary
	var list protocol.PCIList
	if c != nil {
		if resp, err := c.Refresh("devices.summary"); err == nil && resp.OK {
			_ = json.Unmarshal(resp.Observed, &sum)
		}
		if resp, err := c.Refresh("devices.pci"); err == nil && resp.OK {
			_ = json.Unmarshal(resp.Observed, &list)
		}
	}
	summary := fmt.Sprintf("%s %s  PCI %d  USB %d", sum.Vendor, sum.Product, sum.PCICount, sum.USBCount)
	table := uitoolkit.NewTableView([]uitoolkit.TableColumn{
		{Title: "ID"},
		{Title: "Vendor", Width: 90},
		{Title: "Device", Width: 90},
		{Title: "Class", Width: 90},
	}, len(list.Devices), func(row, col int) string {
		if row < 0 || row >= len(list.Devices) {
			return ""
		}
		d := list.Devices[row]
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
		uitoolkit.NewButton("Refresh", func() { win.SetContent(build(a, win, c, page, "refreshed")) }),
	)
}
