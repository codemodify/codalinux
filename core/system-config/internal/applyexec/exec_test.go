package applyexec

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestRefuseUnknown(t *testing.T) {
	r := New()
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{Type: "shell", Mode: "rm -rf /"}}})
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("got %v", err)
	}
}

func TestDisplayScaleEval(t *testing.T) {
	var got []string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		return "ok\n", nil
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpDisplayScale, Output: "Virtual-1", Scale: 2, Mode: "1920x1080@60",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[1] != "eval" {
		t.Fatalf("args %v", got)
	}
	if !strings.Contains(got[2], `hl.monitor({`) || !strings.Contains(got[2], `scale = 2`) {
		t.Fatalf("expr %s", got[2])
	}
	if !strings.Contains(got[2], `output = "Virtual-1"`) {
		t.Fatalf("must use output= like coda-settings, got %s", got[2])
	}
	if strings.Contains(got[2], "name =") {
		t.Fatalf("must not use name=, got %s", got[2])
	}
	if strings.Contains(got[2], "keyword") {
		t.Fatal("must not use keyword")
	}
}

func TestRunRequiresSession(t *testing.T) {
	r := New()
	r.Discover = func() (hyprsession.Session, error) {
		return hyprsession.Session{}, fmt.Errorf("HYPRLAND_INSTANCE_SIGNATURE not set; no instance")
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpDisplayScale, Output: "Virtual-1", Scale: 2,
	}}})
	if err == nil || !strings.Contains(err.Error(), "HYPRLAND_INSTANCE_SIGNATURE") {
		t.Fatalf("got %v", err)
	}
}

func TestHyprctlErrorPropagates(t *testing.T) {
	r := New()
	r.Run = func(string, ...string) (string, error) {
		return "keyword can't work with non-legacy parsers. Use eval.", nil
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpDisplayScale, Output: "eDP-1", Scale: 2,
	}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNetworkWiFiConnectArgv(t *testing.T) {
	var got [][]string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append(got, append([]string{name}, args...))
		return "", nil
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetWiFiConnect, Device: "wlan0", SSID: "Cafe", PSK: "password1",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0][0] != "iwctl" || got[0][len(got[0])-1] != "Cafe" {
		t.Fatalf("%v", got)
	}
	got = nil
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetWiFiConnect, Device: "wlan0", SSID: "HiddenNet", PSK: "password1", Hidden: true,
	}}}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got[0], " ")
	if !strings.Contains(joined, "connect-hidden") {
		t.Fatalf("hidden ssid argv %v", got)
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetWiFiConnect, Device: "wlan0;reboot", SSID: "x",
	}}}); err == nil {
		t.Fatal("bad iface")
	}
}

func TestNetworkdUnitWrite(t *testing.T) {
	r := New()
	r.NetworkDir = t.TempDir()
	var wrote string
	r.WriteFile = func(path string, data []byte, perm os.FileMode) error {
		wrote = string(data)
		return nil
	}
	r.Run = func(string, ...string) (string, error) { return "", nil }
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetIfaceMethod, Device: "enp1s0", Method: "static",
		Address: "10.0.2.15/24", Gateway: "10.0.2.2", DNS: []string{"1.1.1.1"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(wrote, "DHCP=no") || !strings.Contains(wrote, "Address=10.0.2.15/24") {
		t.Fatalf("unit %q", wrote)
	}
}

func boolPtr(v bool) *bool { return &v }

func TestAudioWpctl(t *testing.T) {
	var got []string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		return "", nil
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpAudioVolume, ID: "52", Volume: 0.4,
	}}}); err != nil {
		t.Fatal(err)
	}
	if got[0] != "wpctl" || got[1] != "set-volume" || got[2] != "52" {
		t.Fatalf("%v", got)
	}
}

func TestBluetoothPair(t *testing.T) {
	var got []string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		return "", nil
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpBTPair, Device: "AA:BB:CC:DD:EE:FF",
	}}}); err != nil {
		t.Fatal(err)
	}
	if got[0] != "bluetoothctl" || got[len(got)-2] != "pair" {
		t.Fatalf("%v", got)
	}
	if !strings.Contains(strings.Join(got, " "), "--timeout") {
		t.Fatalf("bluetoothctl must pass --timeout, got %v", got)
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpBTPair, Device: "not-an-addr",
	}}}); err == nil {
		t.Fatal("bad addr")
	}
}

func TestLocaleAndDatetime(t *testing.T) {
	r := New()
	r.LocaleConf = t.TempDir() + "/locale.conf"
	var cmds [][]string
	r.Run = func(name string, args ...string) (string, error) {
		cmds = append(cmds, append([]string{name}, args...))
		return "", nil
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{
		{Type: protocol.OpLocaleLang, Value: "en_US.UTF-8"},
		{Type: protocol.OpDateTimeTimezone, Value: "America/Denver"},
		{Type: protocol.OpDateTimeNTP, Enabled: boolPtr(true)},
	}}); err != nil {
		t.Fatal(err)
	}
	if len(cmds) < 3 {
		t.Fatalf("%v", cmds)
	}
}

func TestNetworkAirplaneRfkill(t *testing.T) {
	var got []string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		return "", nil
	}
	on := true
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetAirplane, Enabled: &on,
	}}}); err != nil {
		t.Fatal(err)
	}
	if got[0] != "rfkill" || got[1] != "block" {
		t.Fatalf("%v", got)
	}
}

func TestNetworkdMergeDropIn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/10-wired.network", []byte("[Match]\nName=enp1s0\n\n[Network]\nDHCP=yes\nIPv6AcceptRA=yes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := New()
	r.NetworkDir = dir
	wrote := map[string]string{}
	r.WriteFile = func(path string, data []byte, perm os.FileMode) error {
		wrote[path] = string(data)
		return nil
	}
	r.Run = func(string, ...string) (string, error) { return "", nil }
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetIfaceMethod, Device: "enp1s0", Method: "static",
		Address: "10.0.2.15/24", Gateway: "10.0.2.2", DNS: []string{"1.1.1.1"}, Search: []string{"lan"},
	}}}); err != nil {
		t.Fatal(err)
	}
	// When Run+WriteFile are set, findNetworkdMatch still reads NetworkDir from disk.
	body := ""
	for p, b := range wrote {
		if strings.Contains(p, "50-coda.conf") || strings.Contains(p, "20-coda-") {
			body = b
		}
	}
	if !strings.Contains(body, "DHCP=no") || !strings.Contains(body, "Domains=lan") {
		t.Fatalf("wrote %#v", wrote)
	}
}

func TestIwdPSKFile(t *testing.T) {
	dir := t.TempDir()
	r := New()
	r.IwdDir = dir
	wrote := map[string]string{}
	r.WriteFile = func(path string, data []byte, perm os.FileMode) error {
		wrote[path] = string(data)
		return nil
	}
	r.Run = func(string, ...string) (string, error) { return "", nil }
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpNetWiFiConnect, Device: "wlan0", SSID: "Cafe", PSK: "password1",
	}}}); err != nil {
		t.Fatal(err)
	}
	body := wrote[dir+"/Cafe.psk"]
	if !strings.Contains(body, "[Security]") || !strings.Contains(body, "Passphrase=password1") {
		t.Fatalf("psk file %q", body)
	}
}

func TestPersistHyprLua(t *testing.T) {
	home := t.TempDir()
	r := New()
	r.Discover = func() (hyprsession.Session, error) {
		return hyprsession.Session{Home: home}, nil
	}
	wrote := map[string]string{}
	r.WriteFile = func(path string, data []byte, perm os.FileMode) error {
		wrote[path] = string(data)
		return nil
	}
	r.Run = func(string, ...string) (string, error) { return "ok\n", nil }
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpDisplayScale, Output: "Virtual-1", Scale: 2, Mode: "1920x1080@60",
	}}}); err != nil {
		t.Fatal(err)
	}
	lua := wrote[home+"/.config/hypr/coda-system-config.lua"]
	if !strings.Contains(lua, `output = "Virtual-1"`) || !strings.Contains(lua, "scale = 2") {
		t.Fatalf("lua %q", lua)
	}
}

func TestInputHyprEval(t *testing.T) {
	var expr string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		if len(args) > 1 {
			expr = args[1]
		}
		return "ok\n", nil
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpInputKBLayout, Value: "us",
	}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(expr, "kb_layout") {
		t.Fatalf("%s", expr)
	}
}
