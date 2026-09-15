package reportprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	p.fillDNS(&m)
	p.fillAirplane(&m)
	p.fillWiFi(&m)
	return rpc.Raw(m), nil
}

func (p *Probe) fillDNS(m *protocol.NetworkModel) {
	resolv := readTrim(p.root("etc/resolv.conf"))
	var dns, search []string
	for _, line := range strings.Split(resolv, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "nameserver":
			dns = append(dns, fields[1])
		case "search", "domain":
			search = append(search, fields[1:]...)
		}
	}
	for i := range m.Links {
		if len(m.Links[i].DNS) == 0 {
			m.Links[i].DNS = append([]string(nil), dns...)
		}
		if len(m.Links[i].Search) == 0 {
			m.Links[i].Search = append([]string(nil), search...)
		}
	}
	p.fillNetworkdMethod(m)
}

func (p *Probe) fillNetworkdMethod(m *protocol.NetworkModel) {
	dir := p.root("etc/systemd/network")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	byName := map[string]*protocol.NetLink{}
	for i := range m.Links {
		byName[m.Links[i].Name] = &m.Links[i]
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".network") {
			continue
		}
		body := readTrim(filepath.Join(dir, e.Name()))
		iface, method, addrs, gw, dns, search := parseNetworkdUnit(body)
		if iface == "" || iface == "*" {
			continue
		}
		l := byName[iface]
		if l == nil {
			continue
		}
		if method != "" {
			l.Method = method
		}
		if gw != "" && l.Gateway == "" {
			l.Gateway = gw
		}
		if len(addrs) > 0 && len(l.Addresses) == 0 {
			l.Addresses = addrs
		}
		if len(dns) > 0 {
			l.DNS = dns
		}
		if len(search) > 0 {
			l.Search = search
		}
	}
}

func parseNetworkdUnit(body string) (iface, method string, addrs []string, gw string, dns, search []string) {
	sec := ""
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			sec = strings.Trim(line, "[]")
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch {
		case strings.EqualFold(sec, "Match") && strings.EqualFold(k, "Name"):
			iface = strings.Fields(v)[0]
		case strings.EqualFold(sec, "Network") && strings.EqualFold(k, "DHCP"):
			if strings.EqualFold(v, "no") || v == "0" {
				method = "static"
			} else {
				method = "dhcp"
			}
		case strings.EqualFold(sec, "Network") && strings.EqualFold(k, "Address"):
			addrs = append(addrs, v)
			if method == "" {
				method = "static"
			}
		case strings.EqualFold(sec, "Network") && strings.EqualFold(k, "Gateway"):
			gw = v
		case strings.EqualFold(sec, "Network") && strings.EqualFold(k, "DNS"):
			dns = append(dns, strings.Fields(v)...)
		case strings.EqualFold(sec, "Network") && strings.EqualFold(k, "Domains"):
			search = append(search, strings.Fields(v)...)
		}
	}
	return iface, method, addrs, gw, dns, search
}

func (p *Probe) fillAirplane(m *protocol.NetworkModel) {
	dir := p.root("sys/class/rfkill")
	ents, err := os.ReadDir(dir)
	if err != nil {
		if raw, err := p.cmd("rfkill", "-J"); err == nil {
			m.Airplane = strings.Contains(raw, `"soft":"blocked"`) && !strings.Contains(raw, `"soft":"unblocked"`)
		}
		return
	}
	blocked, seen := 0, 0
	for _, e := range ents {
		soft := readTrim(p.root("sys/class/rfkill", e.Name(), "soft"))
		if soft == "" {
			continue
		}
		seen++
		if soft == "1" {
			blocked++
		}
	}
	m.Airplane = seen > 0 && blocked == seen
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
		sig := 0
		sec := ""
		nameEnd := len(fields)
		if last := fields[len(fields)-1]; isSignalToken(last) {
			sig = signalQuality(last)
			nameEnd--
		}
		if nameEnd > 0 {
			cand := strings.ToLower(fields[nameEnd-1])
			if cand == "psk" || cand == "open" || cand == "8021x" || cand == "wep" {
				sec = cand
				nameEnd--
			}
		}
		name := fields[0]
		if nameEnd > 0 {
			name = strings.Join(fields[:nameEnd], " ")
		}
		out = append(out, protocol.SSID{SSID: name, Security: sec, Signal: sig})
	}
	return out
}

func isSignalToken(s string) bool {
	if s == "" {
		return false
	}
	if strings.Trim(s, "*") == "" {
		return true
	}
	if _, err := strconv.Atoi(strings.TrimSuffix(s, "dBm")); err == nil {
		return true
	}
	return false
}

func signalQuality(s string) int {
	if stars := strings.Trim(s, "*"); stars == "" && strings.Contains(s, "*") {
		n := strings.Count(s, "*") * 20
		if n > 100 {
			n = 100
		}
		return n
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "dBm"))
	if err != nil {
		return 0
	}
	if n < 0 {
		// dBm typically -30 (good) to -90 (bad)
		q := 2 * (n + 100)
		if q < 0 {
			return 0
		}
		if q > 100 {
			return 100
		}
		return q
	}
	if n > 100 {
		return 100
	}
	return n
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
