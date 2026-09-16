package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/daemon"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestPagesCoverKnownPaths(t *testing.T) {
	got := map[string]bool{}
	for _, p := range pagePaths() {
		got[p] = true
	}
	for _, p := range protocol.KnownPaths {
		if !got[p] {
			t.Fatalf("TUI missing KnownPath %s", p)
		}
	}
	if len(pages) != len(protocol.KnownPaths) {
		t.Fatalf("pages %d known %d", len(pages), len(protocol.KnownPaths))
	}
}

func TestEveryPageBuilds(t *testing.T) {
	s := &session{cli: &client.Client{}, scale: 1, vol: 0.5}
	s.snapshotAll()
	for _, p := range protocol.KnownPaths {
		s.goPath(p)
		pg := s.page()
		if pg.title == "" {
			t.Fatalf("empty title for %s", p)
		}
		if protocol.Settable(p) && len(pg.edits) == 0 {
			t.Fatalf("settable %s has no edit fields", p)
		}
		if !protocol.Settable(p) && len(pg.edits) != 0 {
			t.Fatalf("observe-only %s should not offer edits", p)
		}
	}
}

func TestDirtyPerSection(t *testing.T) {
	s := &session{cli: &client.Client{}, scale: 1, vol: 0.5}
	s.snapshotAll()
	s.goPath(protocol.PathDisplay)
	s.scale = 2
	if !s.dirtyPath(protocol.PathDisplay) {
		t.Fatal("display scale should be dirty")
	}
	if s.dirtyPath(protocol.PathNetwork) || s.dirtyPath(protocol.PathAudio) {
		t.Fatal("display edit must not dirty other sections")
	}
	s.goPath(protocol.PathNetwork)
	if s.dirtyPath(s.path()) {
		t.Fatal("network must stay clean")
	}
}

func TestDirtyObserveOnly(t *testing.T) {
	s := &session{cli: &client.Client{}}
	s.snapshotAll()
	for _, p := range []string{protocol.PathDevicesSummary, protocol.PathDevicesPCI, protocol.PathDevicesUSB, protocol.PathHardwareDMI} {
		if s.dirtyPath(p) {
			t.Fatalf("%s is observe-only", p)
		}
		if _, err := s.desiredJSON(p); err == nil {
			t.Fatalf("%s should refuse desired JSON", p)
		}
		if err := s.applyPath(p); err == nil || !strings.Contains(err.Error(), "observe-only") {
			t.Fatalf("%s apply: %v", p, err)
		}
	}
}

func TestDirtySessionAndStorage(t *testing.T) {
	s := &session{cli: &client.Client{}}
	s.snapshotAll()
	if s.dirtyPath(protocol.PathSession) {
		t.Fatal("session clean on open")
	}
	s.sessionAct = "lock"
	if !s.dirtyPath(protocol.PathSession) {
		t.Fatal("staged lock should dirty")
	}
	s.storageName = "sdb1"
	if s.dirtyPath(protocol.PathStorage) {
		t.Fatal("selecting a disk without action must not dirty")
	}
	s.storageAct = "mount"
	if !s.dirtyPath(protocol.PathStorage) {
		t.Fatal("staged mount should dirty")
	}
}

func TestDesiredJSONSettable(t *testing.T) {
	s := &session{cli: &client.Client{}, scale: 1.5, outName: "Virtual-1", vol: 0.4}
	s.snapshotAll()
	for _, p := range protocol.KnownPaths {
		if !protocol.Settable(p) {
			continue
		}
		if p == protocol.PathSession {
			s.sessionAct = "lock"
		}
		if p == protocol.PathStorage {
			s.storageName = "sdb1"
			s.storageAct = "mount"
		}
		raw, err := s.desiredJSON(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if !json.Valid(raw) {
			t.Fatalf("%s invalid JSON %s", p, raw)
		}
	}
}

func TestDumpListsKnownPaths(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	srv := daemon.New(daemon.Options{
		Socket: sock,
		Scan:   func(string) (json.RawMessage, error) { return json.RawMessage(`{}`), nil },
	})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	go func() { _ = srv.Serve() }()
	c, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	s := &session{cli: c, scale: 1}
	var buf bytes.Buffer
	if err := s.dump(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "not implemented") {
		t.Fatal("dump still a stub")
	}
	for _, p := range protocol.KnownPaths {
		if !strings.Contains(out, p) {
			t.Fatalf("dump missing %s\n%s", p, out)
		}
	}
}

func TestApplyGoesThroughDaemon(t *testing.T) {
	dir := t.TempDir()
	sock := dir + "/d.sock"
	var applied []protocol.PlanOp
	srv := daemon.New(daemon.Options{
		Socket: sock,
		Scan: func(path string) (json.RawMessage, error) {
			if path != protocol.PathDisplay {
				return json.RawMessage(`{}`), nil
			}
			return json.RawMessage(`{"outputs":[{"name":"Virtual-1","scale":1,"mode":"1920x1080@60"}]}`), nil
		},
		Apply: func(p protocol.Plan) error {
			applied = append(applied, p.Ops...)
			return nil
		},
	})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	go func() { _ = srv.Serve() }()
	c, err := client.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	s := &session{cli: c, scale: 1, outName: "Virtual-1"}
	s.snapshotAll()
	s.scale = 2
	if err := s.applyPath(protocol.PathDisplay); err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 || applied[0].Type != protocol.OpDisplayScale || applied[0].Scale != 2 {
		t.Fatalf("plan %+v", applied)
	}
	if s.dirtyPath(protocol.PathDisplay) {
		t.Fatal("display should be clean after apply")
	}
}

func TestNothingToApplyWhenClean(t *testing.T) {
	s := &session{cli: &client.Client{}, scale: 1}
	s.snapshotAll()
	if err := s.applyPath(protocol.PathDisplay); err == nil || !strings.Contains(err.Error(), "nothing to apply") {
		t.Fatalf("got %v", err)
	}
}
