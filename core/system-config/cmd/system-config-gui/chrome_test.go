package main

import (
	"strings"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestNavCoversGUIPaths(t *testing.T) {
	want := []string{
		protocol.PathDisplay, protocol.PathInput, protocol.PathDevicesSummary,
		protocol.PathNetwork, protocol.PathAudio, protocol.PathBluetooth,
		protocol.PathDateTime, protocol.PathLocale, protocol.PathSession,
		protocol.PathPower, protocol.PathPrinters, protocol.PathUsers,
		protocol.PathStorage,
	}
	if len(nav) != len(want) {
		t.Fatalf("nav len %d want %d", len(nav), len(want))
	}
	seen := map[string]bool{}
	for _, n := range nav {
		if n.Label == "" || n.Path == "" || n.Group == "" || n.Blurb == "" {
			t.Fatalf("incomplete nav item %#v", n)
		}
		if seen[n.Path] {
			t.Fatalf("duplicate path %s", n.Path)
		}
		seen[n.Path] = true
		if navIndex(n.Path) < 0 {
			t.Fatalf("navIndex missed %s", n.Path)
		}
		if pageMeta(n.Path).Label != n.Label {
			t.Fatalf("pageMeta(%s) label %q", n.Path, pageMeta(n.Path).Label)
		}
	}
	for _, p := range want {
		if !seen[p] {
			t.Fatalf("nav missing %s", p)
		}
	}
}

func TestNavGroups(t *testing.T) {
	got := navGroups()
	want := []string{"Hardware", "Connectivity", "Session", "System"}
	if len(got) != len(want) {
		t.Fatalf("groups %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("groups %v want %v", got, want)
		}
	}
}

func TestSectionChip(t *testing.T) {
	s := testSess()
	s.cli = nil
	if got := s.sectionChip(protocol.PathDisplay); got == "" {
		t.Fatal("expected disconnected chip")
	}
	s = testSess()
	if got := s.sectionChip(protocol.PathDevicesSummary); got != "Observe only" {
		t.Fatalf("devices chip %q", got)
	}
	if got := s.sectionChip(protocol.PathDisplay); got != "No staged changes" {
		t.Fatalf("clean display chip %q", got)
	}
	s.scale = 2
	if got := s.sectionChip(protocol.PathDisplay); got != "Unapplied changes in this section" {
		t.Fatalf("dirty display chip %q", got)
	}
	if got := s.sectionChip(protocol.PathNetwork); got != "No staged changes" {
		t.Fatalf("display edit must not chip network: %q", got)
	}
}

func TestNavLabelDirtyDot(t *testing.T) {
	s := testSess()
	i := navIndex(protocol.PathDisplay)
	if s.navLabel(i) != "Display" {
		t.Fatalf("clean label %q", s.navLabel(i))
	}
	s.scale = 2
	if s.navLabel(i) != "Display  •" {
		t.Fatalf("dirty label %q", s.navLabel(i))
	}
	net := navIndex(protocol.PathNetwork)
	if s.navLabel(net) != "Network" {
		t.Fatalf("display dirty must not mark network: %q", s.navLabel(net))
	}
}

func TestSectionHintPerSectionApply(t *testing.T) {
	s := testSess()
	h := s.sectionHint(protocol.PathDisplay)
	if !strings.Contains(h, "this section only") {
		t.Fatalf("settable hint should mention per-section apply: %q", h)
	}
	if got := s.sectionHint(protocol.PathDevicesSummary); !strings.Contains(got, "no Apply") {
		t.Fatalf("observe hint %q", got)
	}
}

func TestBuildComposedShell(t *testing.T) {
	s := testSess()
	root := s.build()
	if root == nil {
		t.Fatal("build returned nil")
	}
	if s.navList == nil {
		t.Fatal("nav rail ListView missing")
	}
	if s.navList.Selected != s.page {
		t.Fatalf("rail selected %d page %d", s.navList.Selected, s.page)
	}
	s.goPath(protocol.PathNetwork)
	root = s.build()
	if root == nil || s.navList == nil || s.navList.Selected != navIndex(protocol.PathNetwork) {
		t.Fatalf("rebuild rail selected %v", s.navList)
	}
}
