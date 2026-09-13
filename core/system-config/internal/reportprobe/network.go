package reportprobe

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func (p *Probe) network() (json.RawMessage, error) {
	m := protocol.NetworkModel{}
	byName := map[string]*protocol.NetLink{}

	netDir := p.root("sys/class/net")
	if ents, err := os.ReadDir(netDir); err == nil {
		for _, e := range ents {
			name := e.Name()
			if name == "lo" {
				continue
			}
			oper := readTrim(p.root("sys/class/net", name, "operstate"))
			link := protocol.NetLink{
				Name:      name,
				Type:      "ether",
				OperState: oper,
				Enabled:   oper != "down",
				Method:    "dhcp",
			}
			if _, err := os.Stat(p.root("sys/class/net", name, "wireless")); err == nil {
				link.Type = "wlan"
			}
			m.Links = append(m.Links, link)
			byName[name] = &m.Links[len(m.Links)-1]
		}
	}

	if raw, err := p.cmd("ip", "-j", "addr"); err == nil {
		var addrs []struct {
			Ifname    string `json:"ifname"`
			Operstate string `json:"operstate"`
			AddrInfo  []struct {
				Local     string `json:"local"`
				Prefixlen int    `json:"prefixlen"`
				Family    string `json:"family"`
			} `json:"addr_info"`
		}
		if json.Unmarshal([]byte(raw), &addrs) == nil {
			for _, a := range addrs {
				if a.Ifname == "lo" {
					continue
				}
				l := byName[a.Ifname]
				if l == nil {
					link := protocol.NetLink{Name: a.Ifname, OperState: a.Operstate, Enabled: a.Operstate != "down"}
					m.Links = append(m.Links, link)
					l = &m.Links[len(m.Links)-1]
					byName[a.Ifname] = l
				}
				if a.Operstate != "" {
					l.OperState = a.Operstate
					l.Enabled = a.Operstate != "down"
				}
				for _, ai := range a.AddrInfo {
					if ai.Family != "inet" && ai.Family != "" && ai.Family != "inet6" {
						continue
					}
					if ai.Local == "" {
						continue
					}
					cidr := ai.Local
					if ai.Prefixlen > 0 {
						cidr += "/" + strconv.Itoa(ai.Prefixlen)
					}
					l.Addresses = append(l.Addresses, cidr)
				}
			}
		}
	}

	if raw, err := p.cmd("ip", "-j", "route"); err == nil {
		var routes []struct {
			Dst     string `json:"dst"`
			Gateway string `json:"gateway"`
			Dev     string `json:"dev"`
		}
		if json.Unmarshal([]byte(raw), &routes) == nil {
			for _, rt := range routes {
				m.Routes = append(m.Routes, protocol.Route{Dst: rt.Dst, Gateway: rt.Gateway, Dev: rt.Dev})
				if (rt.Dst == "default" || rt.Dst == "") && rt.Gateway != "" {
					if l := byName[rt.Dev]; l != nil && l.Gateway == "" {
						l.Gateway = rt.Gateway
					}
				}
			}
		}
	}

	p.fillWiFi(&m)
	return rpc.Raw(m), nil
}

func (p *Probe) fillWiFi(m *protocol.NetworkModel) {
	dev := ""
	for _, l := range m.Links {
		if l.Type == "wlan" {
			dev = l.Name
			break
		}
	}
	if raw, err := p.cmd("iwctl", "device", "list"); err == nil {
		if d := parseIwctlDevice(raw); d != "" {
			dev = d
		}
	}
	if dev == "" {
		return
	}
	m.WiFi.Device = dev
	if raw, err := p.cmd("iwctl", "station", dev, "show"); err == nil {
		m.WiFi.Connected = parseIwctlConnected(raw)
	}
	if raw, err := p.cmd("iwctl", "station", dev, "get-networks"); err == nil {
		m.WiFi.Networks = parseIwctlNetworks(raw)
	}
	if raw, err := p.cmd("iwctl", "known-networks", "list"); err == nil {
		m.WiFi.Known = parseIwctlKnown(raw)
	}
}

func stripANSI(s string) string { return ansi.ReplaceAllString(s, "") }

func parseIwctlDevice(raw string) string {
	for _, line := range strings.Split(stripANSI(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] == "Name" || strings.HasPrefix(fields[0], "-") {
			continue
		}
		if strings.HasPrefix(fields[0], "wlan") || strings.HasPrefix(fields[0], "wlp") || strings.HasPrefix(fields[0], "wl") {
			return fields[0]
		}
	}
	return ""
}

func parseIwctlConnected(raw string) string {
	for _, line := range strings.Split(stripANSI(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "connected network") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				return parts[len(parts)-1]
			}
		}
	}
	return ""
}

func parseIwctlNetworks(raw string) []protocol.SSID {
	var out []protocol.SSID
	for _, line := range strings.Split(stripANSI(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Network") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "Available") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		sec := ""
		if len(fields) >= 2 {
			sec = fields[len(fields)-2]
			if sec != "psk" && sec != "open" && sec != "8021x" {
				sec = fields[len(fields)-1]
			}
		}
		name := fields[0]
		if len(fields) > 2 {
			name = strings.Join(fields[:len(fields)-2], " ")
		}
		out = append(out, protocol.SSID{SSID: name, Security: sec})
	}
	return out
}

func parseIwctlKnown(raw string) []string {
	var out []string
	for _, line := range strings.Split(stripANSI(raw), "\n") {
		fields := strings.Fields(stripANSI(line))
		if len(fields) == 0 || fields[0] == "Name" || strings.HasPrefix(fields[0], "-") {
			continue
		}
		out = append(out, fields[0])
	}
	return out
}
