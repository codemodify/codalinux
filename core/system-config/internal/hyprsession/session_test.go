package hyprsession

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverEnv(t *testing.T) {
	root := t.TempDir()
	his := "sig-env"
	runtime := filepath.Join(root, "1000")
	mustInst(t, filepath.Join(runtime, "hypr", his))
	f := &Finder{
		Environ:     []string{"XDG_RUNTIME_DIR=" + runtime, "HYPRLAND_INSTANCE_SIGNATURE=" + his},
		Getuid:      func() int { return 1000 },
		RuntimeRoot: root,
		Lookup:      stubLookup,
		PreferUID:   func() (uint32, bool) { return 0, false },
	}
	s, err := f.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if s.UID != 1000 || s.Signature != his || s.RuntimeDir != runtime {
		t.Fatalf("%+v", s)
	}
}

func TestDiscoverScanPrefersLoginUID(t *testing.T) {
	root := t.TempDir()
	mustInst(t, filepath.Join(root, "1000", "hypr", "live-sig"))
	mustInst(t, filepath.Join(root, "1001", "hypr", "other-sig"))
	f := &Finder{
		Environ:     []string{},
		Getuid:      func() int { return 0 },
		RuntimeRoot: root,
		TmpHypr:     filepath.Join(t.TempDir(), "missing"),
		Lookup:      stubLookup,
		PreferUID:   func() (uint32, bool) { return 1000, true },
	}
	s, err := f.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if s.UID != 1000 || s.Signature != "live-sig" {
		t.Fatalf("want live session, got %+v", s)
	}
}

func TestDiscoverScanAsSessionUser(t *testing.T) {
	root := t.TempDir()
	mustInst(t, filepath.Join(root, "1000", "hypr", "tty1-sig"))
	f := &Finder{
		Environ:     []string{},
		Getuid:      func() int { return 1000 },
		RuntimeRoot: root,
		TmpHypr:     filepath.Join(t.TempDir(), "missing"),
		Lookup:      stubLookup,
		PreferUID:   func() (uint32, bool) { return 0, false },
	}
	s, err := f.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if s.Signature != "tty1-sig" || s.RuntimeDir != filepath.Join(root, "1000") {
		t.Fatalf("%+v", s)
	}
}

func TestDiscoverNone(t *testing.T) {
	root := t.TempDir()
	f := &Finder{
		Environ:     []string{},
		Getuid:      func() int { return 0 },
		RuntimeRoot: root,
		TmpHypr:     filepath.Join(root, "no-hypr"),
		Lookup:      stubLookup,
		PreferUID:   func() (uint32, bool) { return 0, false },
	}
	if _, err := f.Discover(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvironOverlay(t *testing.T) {
	s := Session{
		UID: 1000, User: "live", Home: "/home/live",
		RuntimeDir: "/run/user/1000", Signature: "abc", Wayland: "wayland-1",
	}
	env := s.Environ([]string{"PATH=/usr/bin", "XDG_RUNTIME_DIR=/run/user/0", "FOO=bar"})
	got := map[string]string{}
	for _, e := range env {
		k, v, _ := splitEq(e)
		got[k] = v
	}
	if got["XDG_RUNTIME_DIR"] != "/run/user/1000" {
		t.Fatalf("runtime %q", got["XDG_RUNTIME_DIR"])
	}
	if got["HYPRLAND_INSTANCE_SIGNATURE"] != "abc" {
		t.Fatalf("his %q", got["HYPRLAND_INSTANCE_SIGNATURE"])
	}
	if got["PATH"] != "/usr/bin" || got["FOO"] != "bar" {
		t.Fatalf("%v", got)
	}
	if got["USER"] != "live" || got["HOME"] != "/home/live" {
		t.Fatalf("%v", got)
	}
}

func TestDiscoverEnvHISFindsOtherRuntime(t *testing.T) {
	root := t.TempDir()
	mustInst(t, filepath.Join(root, "1000", "hypr", "from-live"))
	f := &Finder{
		Environ:     []string{"XDG_RUNTIME_DIR=" + filepath.Join(root, "0"), "HYPRLAND_INSTANCE_SIGNATURE=from-live"},
		Getuid:      func() int { return 0 },
		RuntimeRoot: root,
		TmpHypr:     filepath.Join(root, "no-hypr"),
		Lookup:      stubLookup,
		PreferUID:   func() (uint32, bool) { return 1000, true },
	}
	s, err := f.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if s.UID != 1000 || s.RuntimeDir != filepath.Join(root, "1000") {
		t.Fatalf("root HIS should resolve to live runtime, got %+v", s)
	}
}

func stubLookup(uid uint32) (string, string, uint32, []uint32) {
	if uid == 1000 {
		return "live", "/home/live", 1000, []uint32{1000}
	}
	return "u", "/", uid, nil
}

func mustInst(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".socket.sock"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
}

func splitEq(e string) (string, string, bool) {
	for i := 0; i < len(e); i++ {
		if e[i] == '=' {
			return e[:i], e[i+1:], true
		}
	}
	return e, "", false
}
