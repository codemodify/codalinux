package main

import (
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/client"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func testSess() *session {
	s := &session{cli: &client.Client{}, scale: 1, vol: 0.5}
	s.snapshotAll()
	return s
}

func (s *session) goPath(path string) {
	for i, n := range nav {
		if n.Path == path {
			s.page = i
			return
		}
	}
	tpanic := path
	panic("unknown path " + tpanic)
}

func TestDirtyNavAlone(t *testing.T) {
	s := testSess()
	for _, n := range nav {
		s.goPath(n.Path)
		if s.dirty() {
			t.Fatalf("nav to %s enabled dirty with no edits", n.Path)
		}
		if s.dirtyPath(n.Path) {
			t.Fatalf("dirtyPath(%s) true on open", n.Path)
		}
	}
}

func TestDirtyDisplayDoesNotArmNetwork(t *testing.T) {
	s := testSess()
	s.goPath(protocol.PathDisplay)
	s.scale = 2
	if !s.dirtyPath(protocol.PathDisplay) {
		t.Fatal("display scale edit should be dirty")
	}
	if s.dirtyPath(protocol.PathNetwork) || s.dirtyPath(protocol.PathAudio) {
		t.Fatal("display edit must not dirty other sections")
	}
	s.goPath(protocol.PathNetwork)
	if s.dirty() {
		t.Fatal("network Apply must stay off after display edit")
	}
	s.goPath(protocol.PathDisplay)
	if !s.dirty() {
		t.Fatal("display staged state must survive nav")
	}
}

func TestDirtyAudioMute(t *testing.T) {
	s := testSess()
	s.goPath(protocol.PathAudio)
	s.mute = true
	if !s.dirtyPath(protocol.PathAudio) {
		t.Fatal("mute toggle should dirty audio")
	}
	s.snapshot(protocol.PathAudio)
	if s.dirtyPath(protocol.PathAudio) {
		t.Fatal("audio should be clean after snapshot/apply")
	}
}

func TestDirtyObserveOnly(t *testing.T) {
	s := testSess()
	if s.dirtyPath(protocol.PathDevicesSummary) || s.dirtyPath(protocol.PathDevicesPCI) {
		t.Fatal("devices pages are observe-only")
	}
}

func TestDirtyDefaultNotTrue(t *testing.T) {
	s := testSess()
	if s.dirtyPath("not-a-path") {
		t.Fatal("unknown path must not be dirty")
	}
}

func TestDirtySessionNeedsStage(t *testing.T) {
	s := testSess()
	s.goPath(protocol.PathSession)
	if s.dirty() {
		t.Fatal("session must not be dirty on nav")
	}
	s.sessionAct = "lock"
	if !s.dirtyPath(protocol.PathSession) {
		t.Fatal("staged lock should dirty session")
	}
}

func TestDirtyStorageNeedsAction(t *testing.T) {
	s := testSess()
	s.storageName = "sdb1"
	if s.dirtyPath(protocol.PathStorage) {
		t.Fatal("selecting a disk without mount/unmount must not dirty")
	}
	s.storageAct = "mount"
	if !s.dirtyPath(protocol.PathStorage) {
		t.Fatal("staged mount should dirty storage")
	}
}
