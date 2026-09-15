package applyexec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) netIfaceEnable(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	state := "down"
	if op.Enabled != nil && *op.Enabled {
		state = "up"
	}
	out, err := r.runHost("ip", "link", "set", op.Device, state)
	if err != nil {
		return fmt.Errorf("ip link set: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) netIfaceMethod(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	if err := checkMethod(op.Method); err != nil {
		return err
	}
	if err := checkCIDR(op.Address); err != nil {
		return err
	}
	if err := checkIP(op.Gateway); err != nil {
		return err
	}
	for _, d := range op.DNS {
		if err := checkIP(d); err != nil {
			return err
		}
	}
	for _, d := range op.Search {
		if err := checkSearch(d); err != nil {
			return err
		}
	}
	dir := r.NetworkDir
	if dir == "" {
		dir = "/etc/systemd/network"
	}
	override := networkdOverride(op)
	path, body := r.mergeNetworkd(dir, op.Device, override)
	if err := r.writeFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	if out, err := r.runHost("networkctl", "reload"); err != nil {
		return fmt.Errorf("networkctl reload: %w (%s)", err, strings.TrimSpace(out))
	}
	if out, err := r.runHost("networkctl", "reconfigure", op.Device); err != nil {
		return fmt.Errorf("networkctl reconfigure: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) mergeNetworkd(dir, iface, override string) (string, string) {
	match := r.findNetworkdMatch(dir, iface)
	if match != "" && !strings.HasPrefix(match, "20-coda-") {
		drop := filepath.Join(dir, match+".d", "50-coda.conf")
		return drop, override
	}
	path := filepath.Join(dir, "20-coda-"+iface+".network")
	existing := r.readFile(path)
	if existing == "" && match != "" {
		existing = r.readFile(filepath.Join(dir, match))
	}
	if existing == "" {
		return path, "[Match]\nName=" + iface + "\n\n" + override
	}
	return path, mergeNetworkdINI(existing, override)
}

func (r *Runner) findNetworkdMatch(dir, iface string) string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".network") {
			continue
		}
		body := r.readFile(filepath.Join(dir, name))
		if networkdMatches(body, iface) {
			return name
		}
	}
	return ""
}

func (r *Runner) readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func networkdMatches(body, iface string) bool {
	inMatch := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inMatch = strings.EqualFold(line, "[Match]")
			continue
		}
		if !inMatch {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(k), "Name") {
			continue
		}
		for _, n := range strings.Fields(v) {
			if n == iface || n == "*" {
				return true
			}
		}
	}
	return false
}

func networkdOverride(op protocol.PlanOp) string {
	var b strings.Builder
	b.WriteString("[Network]\n")
	if op.Method == "static" {
		b.WriteString("DHCP=no\n")
		if op.Address != "" {
			b.WriteString("Address=")
			b.WriteString(op.Address)
			b.WriteString("\n")
		}
		if op.Gateway != "" {
			b.WriteString("Gateway=")
			b.WriteString(op.Gateway)
			b.WriteString("\n")
		}
	} else {
		b.WriteString("DHCP=yes\n")
	}
	for _, d := range op.DNS {
		b.WriteString("DNS=")
		b.WriteString(d)
		b.WriteString("\n")
	}
	if len(op.Search) > 0 {
		b.WriteString("Domains=")
		b.WriteString(strings.Join(op.Search, " "))
		b.WriteString("\n")
	}
	return b.String()
}

func mergeNetworkdINI(existing, override string) string {
	over := parseINI(override)
	cur := parseINI(existing)
	if cur["Match"] == nil {
		cur["Match"] = map[string][]string{}
	}
	if overNet, ok := over["Network"]; ok {
		if cur["Network"] == nil {
			cur["Network"] = map[string][]string{}
		}
		for k, vs := range overNet {
			cur["Network"][k] = vs
		}
	}
	return writeINI(cur, []string{"Match", "Network", "Link", "Route", "DHCP"})
}

func parseINI(s string) map[string]map[string][]string {
	out := map[string]map[string][]string{}
	sec := ""
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, ";") {
			continue
		}
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			sec = strings.TrimSuffix(strings.TrimPrefix(trim, "["), "]")
			if out[sec] == nil {
				out[sec] = map[string][]string{}
			}
			continue
		}
		if sec == "" {
			continue
		}
		k, v, ok := strings.Cut(trim, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		out[sec][k] = append(out[sec][k], strings.TrimSpace(v))
	}
	return out
}

func writeINI(m map[string]map[string][]string, order []string) string {
	var b strings.Builder
	seen := map[string]bool{}
	writeSec := func(sec string) {
		kv := m[sec]
		if kv == nil {
			return
		}
		if seen[sec] {
			return
		}
		seen[sec] = true
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[")
		b.WriteString(sec)
		b.WriteString("]\n")
		keys := make([]string, 0, len(kv))
		pref := []string{"Name", "DHCP", "Address", "Gateway", "DNS", "Domains"}
		used := map[string]bool{}
		for _, k := range pref {
			if _, ok := kv[k]; ok {
				keys = append(keys, k)
				used[k] = true
			}
		}
		for k := range kv {
			if !used[k] {
				keys = append(keys, k)
			}
		}
		for _, k := range keys {
			for _, v := range kv[k] {
				b.WriteString(k)
				b.WriteString("=")
				b.WriteString(v)
				b.WriteString("\n")
			}
		}
	}
	for _, sec := range order {
		writeSec(sec)
	}
	for sec := range m {
		writeSec(sec)
	}
	return b.String()
}

func (r *Runner) netWiFiConnect(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	if err := checkSSID(op.SSID); err != nil {
		return err
	}
	if err := checkPSK(op.PSK); err != nil {
		return err
	}
	if op.PSK != "" {
		if err := r.writeIwdPSK(op.SSID, op.PSK); err != nil {
			return err
		}
	}
	sub := "connect"
	if op.Hidden {
		sub = "connect-hidden"
	}
	var out string
	var err error
	if op.PSK != "" {
		out, err = r.runHost("iwctl", "--passphrase", op.PSK, "station", op.Device, sub, op.SSID)
	} else {
		out, err = r.runHost("iwctl", "station", op.Device, sub, op.SSID)
	}
	if err != nil {
		return fmt.Errorf("iwctl connect: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) writeIwdPSK(ssid, psk string) error {
	dir := r.IwdDir
	if dir == "" {
		dir = "/var/lib/iwd"
	}
	name := iwdFilename(ssid)
	body := "[Security]\nPassphrase=" + psk + "\n"
	return r.writeFile(filepath.Join(dir, name), []byte(body), 0o600)
}

func iwdFilename(ssid string) string {
	for _, r := range ssid {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			var b strings.Builder
			for _, c := range []byte(ssid) {
				fmt.Fprintf(&b, "%02x", c)
			}
			return b.String() + ".psk"
		}
	}
	return ssid + ".psk"
}

func (r *Runner) netWiFiDisconnect(op protocol.PlanOp) error {
	if err := checkIface(op.Device); err != nil {
		return err
	}
	out, err := r.runHost("iwctl", "station", op.Device, "disconnect")
	if err != nil {
		return fmt.Errorf("iwctl disconnect: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) netAirplane(op protocol.PlanOp) error {
	arg := "unblock"
	if op.Enabled != nil && *op.Enabled {
		arg = "block"
	}
	out, err := r.runHost("rfkill", arg, "all")
	if err != nil {
		return fmt.Errorf("rfkill %s: %w (%s)", arg, err, strings.TrimSpace(out))
	}
	return nil
}
