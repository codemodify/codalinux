package plan

import (
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestFromDisplayScale(t *testing.T) {
	p, err := FromDisplay(
		[]byte(`{"outputs":[{"name":"Virtual-1","scale":2}]}`),
		[]byte(`{"outputs":[{"name":"Virtual-1","width":1920,"height":1080,"refresh_hz":60,"scale":1}]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 1 || p.Ops[0].Scale != 2 || p.Ops[0].Output != "Virtual-1" {
		t.Fatalf("%+v", p.Ops)
	}
}

func TestFromNetworkWifiAndIface(t *testing.T) {
	p, err := FromNetwork(
		[]byte(`{"links":[{"name":"wlan0","enabled":true,"method":"dhcp"}],"wifi":{"device":"wlan0","connect":"Cafe","psk":"secret"}}`),
		[]byte(`{"links":[{"name":"wlan0","enabled":false,"method":"dhcp"}],"wifi":{"device":"wlan0","connected":""}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 {
		t.Fatalf("ops %+v", p.Ops)
	}
	if p.Ops[0].Type != protocol.OpNetIfaceEnable || p.Ops[1].Type != protocol.OpNetWiFiConnect {
		t.Fatalf("%+v", p.Ops)
	}
	if p.Ops[1].SSID != "Cafe" || p.Ops[1].PSK != "secret" {
		t.Fatalf("%+v", p.Ops[1])
	}
}

func TestFromNetworkAirplaneAndDNS(t *testing.T) {
	p, err := FromNetwork(
		[]byte(`{"airplane":true,"links":[{"name":"enp1s0","method":"static","addresses":["10.0.2.15/24"],"gateway":"10.0.2.2","dns":["1.1.1.1"],"search":["lan"]}]}`),
		[]byte(`{"airplane":false,"links":[{"name":"enp1s0","method":"dhcp"}]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 {
		t.Fatalf("ops %+v", p.Ops)
	}
	if p.Ops[0].Type != protocol.OpNetIfaceMethod || p.Ops[0].Search[0] != "lan" {
		t.Fatalf("%+v", p.Ops[0])
	}
	if p.Ops[1].Type != protocol.OpNetAirplane || p.Ops[1].Enabled == nil || !*p.Ops[1].Enabled {
		t.Fatalf("%+v", p.Ops[1])
	}
}

func TestFromDisplayPosition(t *testing.T) {
	p, err := FromDisplay(
		[]byte(`{"outputs":[{"name":"HDMI-A-1","position":"1920x0","scale":1}]}`),
		[]byte(`{"outputs":[{"name":"HDMI-A-1","position":"0x0","scale":1,"x":0,"y":0}]}`),
	)
	if err != nil || len(p.Ops) == 0 || p.Ops[0].Type != protocol.OpDisplayPosition {
		t.Fatalf("%v %+v", err, p.Ops)
	}
}

func TestFromNetworkHiddenSSID(t *testing.T) {
	p, err := FromNetwork(
		[]byte(`{"wifi":{"device":"wlan0","connect":"SecretNet","psk":"password1","hidden":true}}`),
		[]byte(`{"wifi":{"device":"wlan0"}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 1 || !p.Ops[0].Hidden || p.Ops[0].SSID != "SecretNet" {
		t.Fatalf("%+v", p.Ops)
	}
}

func TestFromAudioVolume(t *testing.T) {
	p, err := FromAudio(
		[]byte(`{"default_sink":"52","volume":0.4}`),
		[]byte(`{"default_sink":"40","sinks":[{"id":"40","volume":0.8,"default":true}]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 {
		t.Fatalf("%+v", p.Ops)
	}
}

func TestFromBluetoothPair(t *testing.T) {
	p, err := FromBluetooth(
		[]byte(`{"powered":true,"pair":["AA:BB:CC:DD:EE:FF"]}`),
		[]byte(`{"powered":false}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 || p.Ops[0].Type != protocol.OpBTPower || p.Ops[1].Type != protocol.OpBTPair {
		t.Fatalf("%+v", p.Ops)
	}
}

func TestFromInputLayout(t *testing.T) {
	p, err := FromInput(
		[]byte(`{"kb_layout":"de","natural_scroll":false}`),
		[]byte(`{"kb_layout":"us","natural_scroll":true}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 {
		t.Fatalf("%+v", p.Ops)
	}
}

func TestFromDateTimeAndLocale(t *testing.T) {
	p, err := FromDateTime(
		[]byte(`{"timezone":"America/Denver","ntp":true}`),
		[]byte(`{"timezone":"UTC","ntp":false}`),
	)
	if err != nil || len(p.Ops) != 2 {
		t.Fatalf("%v %+v", err, p.Ops)
	}
	p, err = FromLocale(
		[]byte(`{"lang":"en_US.UTF-8","keymap":"us"}`),
		[]byte(`{"lang":"C","keymap":"de"}`),
	)
	if err != nil || len(p.Ops) != 2 {
		t.Fatalf("%v %+v", err, p.Ops)
	}
}

func TestBuildUnknown(t *testing.T) {
	if _, err := Build("devices.pci", nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestFromSessionLock(t *testing.T) {
	p, err := FromSession([]byte(`{"action":"lock"}`), nil)
	if err != nil || len(p.Ops) != 1 || p.Ops[0].Type != protocol.OpSessionLock {
		t.Fatalf("%v %+v", err, p)
	}
}

func TestFromPowerBrightness(t *testing.T) {
	p, err := FromPower(
		[]byte(`{"brightness":80,"action":"suspend"}`),
		[]byte(`{"brightness":40}`),
	)
	if err != nil || len(p.Ops) != 2 {
		t.Fatalf("%v %+v", err, p.Ops)
	}
}
