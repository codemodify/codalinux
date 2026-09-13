package reportprobe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestDevicesFromSysfs(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "sys/class/dmi/id/sys_vendor"), "QEMU\n")
	mustWrite(t, filepath.Join(root, "sys/class/dmi/id/product_name"), "Standard PC\n")
	mustWrite(t, filepath.Join(root, "sys/bus/pci/devices/0000:00:00.0/vendor"), "0x8086\n")
	mustWrite(t, filepath.Join(root, "sys/bus/pci/devices/0000:00:00.0/device"), "0x1234\n")
	mustWrite(t, filepath.Join(root, "sys/bus/pci/devices/0000:00:00.0/class"), "0x060000\n")
	p := &Probe{Root: root}
	raw, err := p.Collect(protocol.PathDevicesSummary)
	if err != nil {
		t.Fatal(err)
	}
	var sum protocol.DevicesSummary
	if err := json.Unmarshal(raw, &sum); err != nil {
		t.Fatal(err)
	}
	if sum.Vendor != "QEMU" || sum.PCICount != 1 {
		t.Fatalf("%+v", sum)
	}
	raw, err = p.Collect(protocol.PathDevicesPCI)
	if err != nil {
		t.Fatal(err)
	}
	var list protocol.PCIList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Devices) != 1 || list.Devices[0].Vendor != "0x8086" {
		t.Fatalf("%+v", list)
	}
}

func TestDisplayFromHyprctl(t *testing.T) {
	p := &Probe{Hyprctl: func() ([]byte, error) {
		return []byte(`[{"name":"Virtual-1","width":1920,"height":1080,"refreshRate":60,"scale":2,"focused":true}]`), nil
	}}
	raw, err := p.Collect(protocol.PathDisplay)
	if err != nil {
		t.Fatal(err)
	}
	var d protocol.DisplayModel
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Outputs) != 1 || d.Outputs[0].Scale != 2 || d.Outputs[0].Mode != "1920x1080@60" {
		t.Fatalf("%+v", d)
	}
}

func TestDisplayEmptyWithoutSession(t *testing.T) {
	p := &Probe{Discover: func() (hyprsession.Session, error) {
		return hyprsession.Session{}, errors.New("HYPRLAND_INSTANCE_SIGNATURE not set")
	}}
	raw, err := p.Collect(protocol.PathDisplay)
	if err != nil {
		t.Fatal(err)
	}
	var d protocol.DisplayModel
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Outputs) != 0 {
		t.Fatalf("%+v", d)
	}
}

func TestNetworkFromSysfsAndIP(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "sys/class/net/wlan0/operstate"), "up\n")
	if err := os.MkdirAll(filepath.Join(root, "sys/class/net/wlan0/wireless"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &Probe{Root: root, Run: func(name string, args ...string) (string, error) {
		if name == "ip" && len(args) >= 2 && args[1] == "addr" {
			return `[{"ifname":"wlan0","operstate":"UP","addr_info":[{"local":"10.0.0.5","prefixlen":24,"family":"inet"}]}]`, nil
		}
		if name == "ip" && len(args) >= 2 && args[1] == "route" {
			return `[{"dst":"default","gateway":"10.0.0.1","dev":"wlan0"}]`, nil
		}
		if name == "iwctl" && len(args) >= 2 && args[1] == "wlan0" && args[2] == "show" {
			return "Connected network     Cafe\n", nil
		}
		if name == "iwctl" && len(args) >= 1 && args[0] == "device" {
			return "Name\n----\nwlan0  aa:bb  on\n", nil
		}
		return "", nil
	}}
	raw, err := p.Collect(protocol.PathNetwork)
	if err != nil {
		t.Fatal(err)
	}
	var n protocol.NetworkModel
	if err := json.Unmarshal(raw, &n); err != nil {
		t.Fatal(err)
	}
	if len(n.Links) != 1 || n.Links[0].Name != "wlan0" || n.WiFi.Connected != "Cafe" {
		t.Fatalf("%+v", n)
	}
}

func TestUSBAndDMI(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "sys/bus/usb/devices/1-1/idVendor"), "1d6b\n")
	mustWrite(t, filepath.Join(root, "sys/bus/usb/devices/1-1:1.0/bInterfaceClass"), "09\n")
	mustWrite(t, filepath.Join(root, "sys/class/dmi/id/sys_vendor"), "QEMU\n")
	mustWrite(t, filepath.Join(root, "sys/class/dmi/id/product_name"), "Standard PC\n")
	p := &Probe{Root: root}
	raw, err := p.Collect(protocol.PathDevicesUSB)
	if err != nil {
		t.Fatal(err)
	}
	var list protocol.USBList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Devices) != 1 || list.Devices[0].Vendor != "1d6b" {
		t.Fatalf("%+v", list)
	}
	raw, err = p.Collect(protocol.PathHardwareDMI)
	if err != nil {
		t.Fatal(err)
	}
	var d protocol.DMI
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if d.Vendor != "QEMU" {
		t.Fatalf("%+v", d)
	}
}

func TestParseIwctlAndWpctl(t *testing.T) {
	nets := parseIwctlNetworks("Network name Security Signal\n----\nCafe psk ****\n")
	if len(nets) != 1 || nets[0].SSID != "Cafe" || nets[0].Security != "psk" || nets[0].Signal != 80 {
		t.Fatalf("%+v", nets)
	}
	v, m := parseWpVolume("Volume: 0.40 [MUTED]\n")
	if v != 0.4 || m == nil || !*m {
		t.Fatalf("%v %v", v, m)
	}
	var audio protocol.AudioModel
	parseWpStatus("Audio\n ├─ Sinks:\n │  *   52. Built-in Audio Analog Stereo [vol: 0.50]\n ├─ Sources:\n │      53. Mic [vol: 0.20]\n", &audio)
	if audio.DefaultSink != "52" || len(audio.Sinks) != 1 || audio.Sources[0].ID != "53" {
		t.Fatalf("%+v", audio)
	}
}

func TestPowerFromSysfs(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "sys/class/backlight/acpi_video0/brightness"), "80\n")
	mustWrite(t, filepath.Join(root, "sys/class/backlight/acpi_video0/max_brightness"), "100\n")
	mustWrite(t, filepath.Join(root, "sys/power/state"), "freeze mem disk\n")
	p := &Probe{Root: root, Run: func(string, ...string) (string, error) {
		return "", errStr("no busctl")
	}}
	raw, err := p.Collect(protocol.PathPower)
	if err != nil {
		t.Fatal(err)
	}
	var pw protocol.PowerModel
	if err := json.Unmarshal(raw, &pw); err != nil {
		t.Fatal(err)
	}
	if pw.Backlight != "acpi_video0" || pw.Brightness != 80 || !pw.CanSuspend || !pw.CanHibernate {
		t.Fatalf("%+v", pw)
	}
}

func TestUsersPrintersStorage(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "etc/passwd"), "root:x:0:0:root:/root:/bin/bash\nnobody:x:65534:65534:n:/:/usr/bin/nologin\nlive:x:1000:1000:Live:/home/live:/bin/bash\n")
	p := &Probe{Root: root, Run: func(name string, args ...string) (string, error) {
		if name == "lpstat" {
			return "", errStr("no cups")
		}
		if name == "lsblk" {
			return `{"blockdevices":[{"name":"vda","type":"disk","size":"8G","mountpoint":"","model":"QEMU","children":[{"name":"vda1","type":"part","size":"8G","mountpoint":"/"}]}]}`, nil
		}
		return "", errStr("no")
	}}
	raw, err := p.Collect(protocol.PathUsers)
	if err != nil {
		t.Fatal(err)
	}
	var u protocol.UsersModel
	if err := json.Unmarshal(raw, &u); err != nil {
		t.Fatal(err)
	}
	if len(u.Users) != 2 || u.Users[1].Name != "live" {
		t.Fatalf("%+v", u)
	}
	raw, err = p.Collect(protocol.PathPrinters)
	if err != nil {
		t.Fatal(err)
	}
	var pr protocol.PrintersModel
	if err := json.Unmarshal(raw, &pr); err != nil || len(pr.Printers) != 0 {
		t.Fatalf("%v %+v", err, pr)
	}
	raw, err = p.Collect(protocol.PathStorage)
	if err != nil {
		t.Fatal(err)
	}
	var st protocol.StorageModel
	if err := json.Unmarshal(raw, &st); err != nil || len(st.Block) != 2 {
		t.Fatalf("%v %+v", err, st)
	}
}

func TestBluetoothEmptyOnTimeout(t *testing.T) {
	p := &Probe{Run: func(name string, args ...string) (string, error) {
		return "", errStr("timeout after 2.5s running bluetoothctl")
	}}
	raw, err := p.Collect(protocol.PathBluetooth)
	if err != nil {
		t.Fatal(err)
	}
	var bt protocol.BluetoothModel
	if err := json.Unmarshal(raw, &bt); err != nil {
		t.Fatal(err)
	}
	if bt.Adapter != "" || len(bt.Devices) != 0 {
		t.Fatalf("%+v", bt)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
