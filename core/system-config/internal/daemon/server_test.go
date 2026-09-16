package daemon

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
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
	if strings.Contains(resp.Note, "stub") || !strings.Contains(resp.Note, "follow") {
		t.Fatalf("watch note should document follow stream, got %q", resp.Note)
	}

	resp, err = c.Set("network", json.RawMessage(`{"wifi":{"device":"wlan0","connect":"Cafe"}}`))
	if err != nil || !resp.OK {
		t.Fatalf("set network: %v %+v", err, resp)
	}
	resp, err = c.Apply("network")
	if err != nil || !resp.OK {
		t.Fatalf("apply network: %v %+v", err, resp)
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

func TestWatchFollowObservedStream(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	var n atomic.Int64
	s := New(Options{
		Socket: sock,
		Scan: func(string) (json.RawMessage, error) {
			return json.RawMessage(`{"n":` + strconv.FormatInt(n.Add(1), 10) + `}`), nil
		},
	})
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() { _ = s.Serve() }()

	w, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	snap, err := w.WatchFollow("display")
	if err != nil || !snap.OK {
		t.Fatalf("watch follow: %v %+v", err, snap)
	}
	if snap.Note != protocol.WatchNoteFollow {
		t.Fatalf("first note %q", snap.Note)
	}

	other, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if resp, err := other.Refresh("display"); err != nil || !resp.OK {
		t.Fatalf("refresh: %v %+v", err, resp)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ev, err := w.Next(ctx)
	if err != nil || !ev.OK {
		t.Fatalf("next: %v %+v", err, ev)
	}
	if ev.Note != protocol.WatchNoteObserved || ev.Path != protocol.PathDisplay {
		t.Fatalf("event %+v", ev)
	}
	if !strings.Contains(string(ev.Observed), `"n":`) {
		t.Fatalf("observed %s", ev.Observed)
	}
}

func TestWatchFollowDesiredAndFilter(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	s := New(Options{
		Socket: sock,
		Scan:   func(string) (json.RawMessage, error) { return json.RawMessage(`{}`), nil },
	})
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() { _ = s.Serve() }()

	w, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.WatchFollow("audio"); err != nil {
		t.Fatal(err)
	}

	other, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if resp, err := other.Set("display", json.RawMessage(`{"outputs":[]}`)); err != nil || !resp.OK {
		t.Fatalf("set display: %v %+v", err, resp)
	}
	if resp, err := other.Set("audio", json.RawMessage(`{"volume":0.4}`)); err != nil || !resp.OK {
		t.Fatalf("set audio: %v %+v", err, resp)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ev, err := w.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Path != protocol.PathAudio || ev.Note != protocol.WatchNoteDesired {
		t.Fatalf("expected audio desired, got %+v", ev)
	}
}

func TestWatchFollowPutObserved(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	s := New(Options{Socket: sock, Scan: func(string) (json.RawMessage, error) { return json.RawMessage(`{}`), nil }})
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	go func() { _ = s.Serve() }()

	w, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err := w.WatchFollow(""); err != nil {
		t.Fatal(err)
	}

	other, err := rpc.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	resp, err := other.Call(protocol.Request{Op: protocol.OpPutObserved, Path: "storage", Data: json.RawMessage(`{"block":[{"name":"sda"}]}`)})
	if err != nil || !resp.OK {
		t.Fatalf("put-observed: %v %+v", err, resp)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ev, err := w.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Path != protocol.PathStorage || ev.Note != protocol.WatchNoteObserved {
		t.Fatalf("event %+v", ev)
	}
}
