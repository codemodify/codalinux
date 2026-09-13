package applyexec

import (
	"os"
	"strings"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestPrinterUserStorageArgv(t *testing.T) {
	var got [][]string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append(got, append([]string{name}, args...))
		return "", nil
	}
	on := true
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{
		{Type: protocol.OpPrinterDefault, Name: "HP-Laser"},
		{Type: protocol.OpPrinterEnable, Name: "HP-Laser", Enabled: &on},
		{Type: protocol.OpUserShell, Name: "live", Value: "/bin/bash"},
		{Type: protocol.OpStorageMount, Device: "sdb1"},
	}}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("%v", got)
	}
	if got[0][0] != "lpadmin" || got[0][1] != "-d" || got[0][2] != "HP-Laser" {
		t.Fatalf("lpadmin %v", got[0])
	}
	if got[1][0] != "cupsenable" {
		t.Fatalf("enable %v", got[1])
	}
	if got[2][0] != "usermod" || got[2][1] != "-s" || got[2][2] != "/bin/bash" || got[2][3] != "live" {
		t.Fatalf("usermod %v", got[2])
	}
	if got[3][0] != "udisksctl" || got[3][1] != "mount" || got[3][3] != "/dev/sdb1" {
		t.Fatalf("mount %v", got[3])
	}
}

func TestStorageRefuseSystemMount(t *testing.T) {
	r := New()
	r.Run = func(string, ...string) (string, error) {
		t.Fatal("must not run udisksctl for system unmount")
		return "", nil
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpStorageUnmount, Device: "vda2", Value: "/",
	}}})
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("got %v", err)
	}
}

func TestRefuseBadPrinterUserShellBlock(t *testing.T) {
	r := New()
	r.Run = func(string, ...string) (string, error) { return "", nil }
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{Type: protocol.OpPrinterDefault, Name: "bad name"}}}); err == nil {
		t.Fatal("bad printer")
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{Type: protocol.OpUserShell, Name: "root;reboot", Value: "/bin/bash"}}}); err == nil {
		t.Fatal("bad user")
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{Type: protocol.OpUserShell, Name: "live", Value: "/bin/bash;id"}}}); err == nil {
		t.Fatal("bad shell")
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{Type: protocol.OpStorageMount, Device: "../sda"}}}); err == nil {
		t.Fatal("bad block")
	}
}

func TestAudioPersistWireplumber(t *testing.T) {
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
	r.Run = func(name string, args ...string) (string, error) {
		if name == "wpctl" && len(args) >= 1 && args[0] == "inspect" {
			return "id 52\n  * node.name = \"alsa_output.pci-0000_00_1f.3\"\n", nil
		}
		return "", nil
	}
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpAudioDefaultSink, ID: "52",
	}}}); err != nil {
		t.Fatal(err)
	}
	body := wrote[home+"/.config/wireplumber/wireplumber.conf.d/51-coda-defaults.conf"]
	if !strings.Contains(body, "# coda-sink=alsa_output.pci-0000_00_1f.3") {
		t.Fatalf("marker %q", body)
	}
	if !strings.Contains(body, "monitor.alsa.rules") || !strings.Contains(body, "priority.session = 2000") {
		t.Fatalf("rule %q", body)
	}
	n := strings.Count(body, "monitor.alsa.rules")
	if n != 1 {
		t.Fatalf("want one rules array, got %d in %q", n, body)
	}
}

func TestBluetoothPINFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	r := New()
	wrote := map[string]string{}
	r.WriteFile = func(path string, data []byte, perm os.FileMode) error {
		wrote[path] = string(data)
		return nil
	}
	r.Run = func(string, ...string) (string, error) { return "", nil }
	if err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpBTPair, Device: "AA:BB:CC:DD:EE:FF", PIN: "123456",
	}}}); err != nil {
		t.Fatal(err)
	}
	body := wrote[dir+"/coda/bluetooth-pin"]
	if strings.TrimSpace(body) != "123456" {
		t.Fatalf("pin file %q", body)
	}
}

func TestCheckBlockNames(t *testing.T) {
	for _, ok := range []string{"sda", "sdb1", "vda2", "nvme0n1", "nvme0n1p2", "mmcblk0p1"} {
		if err := checkBlock(ok); err != nil {
			t.Fatalf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"../sda", "sda/../sdb", "sda;reboot", ""} {
		if err := checkBlock(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}
