// Package hyprsession finds the active Hyprland user session so root apply/report
// can run hyprctl with that user's XDG_RUNTIME_DIR and HYPRLAND_INSTANCE_SIGNATURE.
// Mirrors scripts/coda-settings (normal user session) and scripts/coda-sync-desktop-from-host.sh.
package hyprsession

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// Session is one Hyprland instance (typically the seat0 graphical user).
type Session struct {
	UID        uint32
	GID        uint32
	Groups     []uint32
	User       string
	Home       string
	RuntimeDir string
	Signature  string
	Wayland    string
}

// Finder is the testable discovery surface. Zero value uses the host.
type Finder struct {
	Environ     []string
	Getuid      func() int
	RuntimeRoot string // default /run/user
	TmpHypr     string // default /tmp/hypr
	Lookup      func(uid uint32) (name, home string, gid uint32, groups []uint32)
	PreferUID   func() (uint32, bool) // loginctl graphical session; nil → default
}

func Discover() (Session, error) {
	return (&Finder{}).Discover()
}

// DiscoverRuntime finds the graphical user's XDG_RUNTIME_DIR even when Hyprland
// is missing (PipeWire / session-bus tools). Prefers a Hyprland instance.
func DiscoverRuntime() (Session, error) {
	return (&Finder{}).DiscoverRuntime()
}

func (f *Finder) DiscoverRuntime() (Session, error) {
	if s, err := f.Discover(); err == nil {
		return s, nil
	}
	uid, ok := f.preferUID()
	if !ok {
		if u := f.uid(); u != 0 {
			uid, ok = uint32(u), true
		}
	}
	if !ok {
		ents, err := os.ReadDir(f.runtimeRoot())
		if err != nil {
			return Session{}, fmt.Errorf("no graphical user session")
		}
		for _, e := range ents {
			id, err := strconv.ParseUint(e.Name(), 10, 32)
			if err != nil || id == 0 {
				continue
			}
			uid, ok = uint32(id), true
			break
		}
	}
	if !ok || uid == 0 {
		return Session{}, fmt.Errorf("no graphical user session")
	}
	rt := filepath.Join(f.runtimeRoot(), strconv.FormatUint(uint64(uid), 10))
	if x := f.getenv("XDG_RUNTIME_DIR"); x != "" && f.uid() != 0 {
		rt = x
	}
	return f.make(uid, rt, f.getenv("HYPRLAND_INSTANCE_SIGNATURE")), nil
}

func (f *Finder) Discover() (Session, error) {
	his := f.getenv("HYPRLAND_INSTANCE_SIGNATURE")
	runtime := f.getenv("XDG_RUNTIME_DIR")
	uid := uint32(f.uid())
	if runtime == "" {
		runtime = filepath.Join(f.runtimeRoot(), strconv.FormatUint(uint64(uid), 10))
	}

	if his != "" {
		if instDirOK(filepath.Join(runtime, "hypr", his)) {
			return f.make(uid, runtime, his), nil
		}
		for _, c := range f.scan() {
			if c.Signature == his {
				return f.make(c.UID, c.RuntimeDir, c.Signature), nil
			}
		}
		// Trust the caller (same as a normal Hyprland user shell).
		return f.make(uid, runtime, his), nil
	}

	cands := f.scan()
	if len(cands) == 0 {
		return Session{}, fmt.Errorf("HYPRLAND_INSTANCE_SIGNATURE not set; no instance under %s/*/hypr or %s (is Hyprland running?)", f.runtimeRoot(), f.tmpHypr())
	}

	prefer, ok := f.preferUID()
	if !ok {
		if u := f.uid(); u != 0 {
			prefer, ok = uint32(u), true
		}
	}
	best := pick(cands, prefer, ok)
	return f.make(best.UID, best.RuntimeDir, best.Signature), nil
}

type cand struct {
	UID        uint32
	RuntimeDir string
	Signature  string
	Socket     bool
}

func (f *Finder) scan() []cand {
	var out []cand
	root := f.runtimeRoot()
	ents, err := os.ReadDir(root)
	if err == nil {
		for _, e := range ents {
			id, err := strconv.ParseUint(e.Name(), 10, 32)
			if err != nil {
				continue
			}
			runtime := filepath.Join(root, e.Name())
			out = append(out, listHypr(uint32(id), runtime)...)
		}
	}
	// Always include the process uid even if /run/user listing failed.
	uid := uint32(f.uid())
	self := filepath.Join(f.runtimeRoot(), strconv.FormatUint(uint64(uid), 10))
	if f.getenv("XDG_RUNTIME_DIR") != "" {
		self = f.getenv("XDG_RUNTIME_DIR")
	}
	out = append(out, listHypr(uid, self)...)

	tmp := f.tmpHypr()
	prefer, ok := f.preferUID()
	tmpUID := uid
	if ok {
		tmpUID = prefer
	}
	if ents, err := os.ReadDir(tmp); err == nil {
		rt := filepath.Join(f.runtimeRoot(), strconv.FormatUint(uint64(tmpUID), 10))
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(tmp, e.Name())
			out = append(out, cand{
				UID: tmpUID, RuntimeDir: rt, Signature: e.Name(), Socket: instDirOK(dir),
			})
		}
	}
	return uniq(out)
}

func listHypr(uid uint32, runtime string) []cand {
	hypr := filepath.Join(runtime, "hypr")
	ents, err := os.ReadDir(hypr)
	if err != nil {
		return nil
	}
	var out []cand
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(hypr, e.Name())
		out = append(out, cand{
			UID: uid, RuntimeDir: runtime, Signature: e.Name(), Socket: instDirOK(dir),
		})
	}
	return out
}

func instDirOK(dir string) bool {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return false
	}
	for _, name := range []string{".socket.sock", ".socket2.sock", "socket", "socket.sock"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return true // dir exists; hyprctl can still use HIS
	}
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".sock") {
			return true
		}
	}
	return true
}

func pick(cands []cand, prefer uint32, havePrefer bool) cand {
	best := cands[0]
	score := func(c cand) int {
		n := 0
		if c.Socket {
			n += 2
		}
		if havePrefer && c.UID == prefer {
			n += 4
		}
		return n
	}
	bestN := score(best)
	for _, c := range cands[1:] {
		if n := score(c); n > bestN {
			best, bestN = c, n
		}
	}
	return best
}

func uniq(in []cand) []cand {
	type key struct {
		uid uint32
		rt  string
		sig string
	}
	seen := map[key]bool{}
	var out []cand
	for _, c := range in {
		if c.Signature == "" {
			continue
		}
		k := key{c.UID, c.RuntimeDir, c.Signature}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, c)
	}
	return out
}

func (f *Finder) make(uid uint32, runtime, sig string) Session {
	s := Session{UID: uid, RuntimeDir: runtime, Signature: sig, Wayland: waylandDisplay(runtime)}
	if f.Lookup != nil {
		s.User, s.Home, s.GID, s.Groups = f.Lookup(uid)
		return s
	}
	s.User, s.Home, s.GID, s.Groups = defaultLookup(uid)
	return s
}

func defaultLookup(uid uint32) (name, home string, gid uint32, groups []uint32) {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return "", "", uid, nil
	}
	name = u.Username
	home = u.HomeDir
	if n, err := strconv.ParseUint(u.Gid, 10, 32); err == nil {
		gid = uint32(n)
	}
	if ids, err := u.GroupIds(); err == nil {
		for _, id := range ids {
			if n, err := strconv.ParseUint(id, 10, 32); err == nil {
				groups = append(groups, uint32(n))
			}
		}
	}
	return name, home, gid, groups
}

func (f *Finder) preferUID() (uint32, bool) {
	if f.PreferUID != nil {
		return f.PreferUID()
	}
	return loginCtlUID()
}

func loginCtlUID() (uint32, bool) {
	out, err := exec.Command("loginctl", "list-sessions", "--no-legend", "--no-pager").Output()
	if err != nil {
		return 0, false
	}
	var fallback uint32
	var haveFB bool
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		sid := fields[0]
		typ, class, remote, uid := showSession(sid)
		if uid == 0 && len(fields) >= 2 {
			if n, err := strconv.ParseUint(fields[1], 10, 32); err == nil {
				uid = uint32(n)
			}
		}
		if uid == 0 {
			continue
		}
		if class != "" && class != "user" {
			continue
		}
		if remote == "yes" {
			continue
		}
		if typ == "wayland" {
			return uid, true
		}
		if !haveFB {
			fallback, haveFB = uid, true
		}
	}
	return fallback, haveFB
}

func showSession(id string) (typ, class, remote string, uid uint32) {
	out, err := exec.Command("loginctl", "show-session", id, "-p", "Type", "-p", "Class", "-p", "Remote", "-p", "UID").Output()
	if err != nil {
		return "", "", "", 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch k {
		case "Type":
			typ = v
		case "Class":
			class = v
		case "Remote":
			remote = v
		case "UID":
			n, err := strconv.ParseUint(v, 10, 32)
			if err == nil {
				uid = uint32(n)
			}
		}
	}
	return typ, class, remote, uid
}

func waylandDisplay(runtime string) string {
	ents, err := os.ReadDir(runtime)
	if err != nil {
		return "wayland-1"
	}
	for _, e := range ents {
		n := e.Name()
		if strings.HasPrefix(n, "wayland-") && !strings.HasSuffix(n, ".lock") {
			return n
		}
	}
	return "wayland-1"
}

func (f *Finder) getenv(k string) string {
	env := f.Environ
	if env == nil {
		return os.Getenv(k)
	}
	for i := len(env) - 1; i >= 0; i-- {
		e := env[i]
		if strings.HasPrefix(e, k+"=") {
			return e[len(k)+1:]
		}
	}
	return ""
}

func (f *Finder) uid() int {
	if f.Getuid != nil {
		return f.Getuid()
	}
	return os.Getuid()
}

func (f *Finder) runtimeRoot() string {
	if f.RuntimeRoot != "" {
		return f.RuntimeRoot
	}
	return "/run/user"
}

func (f *Finder) tmpHypr() string {
	if f.TmpHypr != "" {
		return f.TmpHypr
	}
	return "/tmp/hypr"
}

// Command is hyprctl (or any helper) with session env, and as the session uid when we are root.
func (s Session) Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env = s.Environ(os.Environ())
	applySessionCreds(cmd, s)
	return cmd
}

// Environ overlays Hyprland session variables on the parent environment.
func (s Session) Environ(base []string) []string {
	if base == nil {
		base = os.Environ()
	}
	env := append([]string{}, base...)
	set := func(k, v string) {
		if v == "" {
			return
		}
		pref := k + "="
		for i, e := range env {
			if strings.HasPrefix(e, pref) {
				env[i] = pref + v
				return
			}
		}
		env = append(env, pref+v)
	}
	set("XDG_RUNTIME_DIR", s.RuntimeDir)
	set("HYPRLAND_INSTANCE_SIGNATURE", s.Signature)
	set("WAYLAND_DISPLAY", s.Wayland)
	set("XDG_SESSION_TYPE", "wayland")
	set("XDG_CURRENT_DESKTOP", "Hyprland")
	set("HOME", s.Home)
	set("USER", s.User)
	set("LOGNAME", s.User)
	return env
}

func (s Session) String() string {
	return fmt.Sprintf("uid=%d user=%s XDG_RUNTIME_DIR=%s HYPRLAND_INSTANCE_SIGNATURE=%s",
		s.UID, s.User, s.RuntimeDir, s.Signature)
}
