package plan

import (
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestFromPrintersDefaultAndEnable(t *testing.T) {
	on := true
	_ = on
	p, err := FromPrinters(
		[]byte(`{"default":"HP","printers":[{"name":"HP","enabled":true}]}`),
		[]byte(`{"default":"Other","printers":[{"name":"HP","enabled":false}]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 {
		t.Fatalf("ops %+v", p.Ops)
	}
	if p.Ops[0].Type != protocol.OpPrinterDefault || p.Ops[0].Name != "HP" {
		t.Fatalf("%+v", p.Ops[0])
	}
	if p.Ops[1].Type != protocol.OpPrinterEnable || p.Ops[1].Name != "HP" || p.Ops[1].Enabled == nil || !*p.Ops[1].Enabled {
		t.Fatalf("%+v", p.Ops[1])
	}
}

func TestFromUsersShell(t *testing.T) {
	p, err := FromUsers(
		[]byte(`{"users":[{"name":"live","shell":"/bin/zsh"}]}`),
		[]byte(`{"users":[{"name":"live","shell":"/bin/bash"}]}`),
	)
	if err != nil || len(p.Ops) != 1 || p.Ops[0].Type != protocol.OpUserShell || p.Ops[0].Value != "/bin/zsh" {
		t.Fatalf("%v %+v", err, p.Ops)
	}
}

func TestFromUsersSameShellNoOp(t *testing.T) {
	p, err := FromUsers(
		[]byte(`{"users":[{"name":"live","shell":"/bin/bash"}]}`),
		[]byte(`{"users":[{"name":"live","shell":"/bin/bash"}]}`),
	)
	if err != nil || len(p.Ops) != 0 {
		t.Fatalf("%v %+v", err, p.Ops)
	}
}

func TestFromStorageMountUnmount(t *testing.T) {
	p, err := FromStorage(
		[]byte(`{"block":[{"name":"sdb1","action":"mount"},{"name":"sdc1","action":"unmount","mount":"/run/media/live/USB"}]}`),
		nil,
	)
	if err != nil || len(p.Ops) != 2 {
		t.Fatalf("%v %+v", err, p.Ops)
	}
	if p.Ops[0].Type != protocol.OpStorageMount || p.Ops[0].Device != "sdb1" {
		t.Fatalf("%+v", p.Ops[0])
	}
	if p.Ops[1].Type != protocol.OpStorageUnmount || p.Ops[1].Value != "/run/media/live/USB" {
		t.Fatalf("%+v", p.Ops[1])
	}
}

func TestBuildPrintersUsersStorage(t *testing.T) {
	if _, err := Build("printers", []byte(`{"default":"HP"}`), []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := Build("users", []byte(`{"users":[{"name":"live","shell":"/bin/bash"}]}`), []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := Build("storage", []byte(`{"block":[{"name":"sdb1","action":"mount"}]}`), []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
}
