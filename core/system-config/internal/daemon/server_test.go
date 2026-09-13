package daemon

import (
	"encoding/json"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestGetSetRefreshApply(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	var applied []protocol.PlanOp
	s := New(Options{
		Socket: sock,
		Scan: func(path string) (json.RawMessage, error) {
			if path != protocol.PathDisplay {
				return json.RawMessage(`{}`), nil
			}
			return json.RawMessage(`{"outputs":[{"name":"Virtual-1","width":1920,"height":1080,"refresh_hz":60,"scale":1,"mode":"1920x1080@60"}]}`), nil
		},
		Apply: func(p protocol.Plan) error {
			applied = append(applied, p.Ops...)
			return nil
		},
	})
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() { _ = s.Serve() }()

	c, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Get("submodels")
	if err != nil || !resp.OK {
		t.Fatalf("get submodels: %v %+v", err, resp)
	}
	resp, err = c.Refresh("display")
	if err != nil || !resp.OK {
		t.Fatalf("refresh: %v %+v", err, resp)
	}
	resp, err = c.Set("display", json.RawMessage(`{"outputs":[{"name":"Virtual-1","scale":2}]}`))
	if err != nil || !resp.OK {
		t.Fatalf("set: %v %+v", err, resp)
	}
	if resp.Status == nil || !resp.Status.Changed {
		t.Fatalf("expected changed %+v", resp.Status)
	}
	resp, err = c.Apply("display")
	if err != nil || !resp.OK {
		t.Fatalf("apply: %v %+v", err, resp)
	}
	if len(applied) == 0 || applied[0].Type != protocol.OpDisplayScale || applied[0].Scale != 2 {
		t.Fatalf("plan %+v", applied)
	}
	resp, err = c.Watch("display")
	if err != nil || !resp.OK || resp.Note == "" {
		t.Fatalf("watch: %v %+v", err, resp)
	}
}

func TestRefuseUnknownSet(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	s := New(Options{Socket: sock, Scan: func(string) (json.RawMessage, error) { return json.RawMessage(`{}`), nil }})
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() { _ = s.Serve() }()
	c, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	resp, err := c.Set("devices.pci", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Fatal("set devices.pci should fail")
	}
}
