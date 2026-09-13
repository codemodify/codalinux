// Package applyexec is the closed allowlist for system-config-apply.
// No arbitrary shell. Typed argv only; user strings are validated first.
package applyexec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/runcmd"
)

type Runner struct {
	Hyprctl       string
	LookPath      func(string) (string, error)
	Run           func(name string, args ...string) (string, error)
	Discover      func() (hyprsession.Session, error)
	WriteFile     func(path string, data []byte, perm os.FileMode) error
	NetworkDir    string
	LocaleConf    string
	VConsole      string
	LogindDrop    string
	BacklightRoot string
	IwdDir        string
	Timeout       time.Duration
}

func New() *Runner {
	return &Runner{Hyprctl: "hyprctl"}
}

func (r *Runner) Exec(p protocol.Plan) error {
	for i, op := range p.Ops {
		if err := r.execOp(op); err != nil {
			return fmt.Errorf("op %d %s: %w", i, op.Type, err)
		}
	}
	return nil
}

func (r *Runner) execOp(op protocol.PlanOp) error {
	if !protocol.AllowedOp(op.Type) {
		return fmt.Errorf("refused: unknown op %q", op.Type)
	}
	switch op.Type {
	case protocol.OpDisplayScale, protocol.OpDisplayMode, protocol.OpDisplayPosition:
		return r.hyprMonitor(op)
	case protocol.OpNetIfaceEnable:
		return r.netIfaceEnable(op)
	case protocol.OpNetIfaceMethod:
		return r.netIfaceMethod(op)
	case protocol.OpNetWiFiConnect:
		return r.netWiFiConnect(op)
	case protocol.OpNetWiFiDisconnect:
		return r.netWiFiDisconnect(op)
	case protocol.OpNetAirplane:
		return r.netAirplane(op)
	case protocol.OpAudioDefaultSink, protocol.OpAudioDefaultSource:
		return r.audioDefault(op)
	case protocol.OpAudioVolume:
		return r.audioVolume(op)
	case protocol.OpAudioMute:
		return r.audioMute(op)
	case protocol.OpBTPower:
		return r.btPower(op)
	case protocol.OpBTScan:
		return r.btScan(op)
	case protocol.OpBTPair, protocol.OpBTConnect, protocol.OpBTDisconnect, protocol.OpBTTrust:
		return r.btDevice(op)
	case protocol.OpInputKeymap:
		return r.inputKeymap(op)
	case protocol.OpInputKBLayout:
		return r.inputKBLayout(op)
	case protocol.OpInputPointerSpeed, protocol.OpInputNaturalScroll, protocol.OpInputTapToClick:
		return r.inputHypr(op)
	case protocol.OpDateTimeTimezone:
		return r.dateTimezone(op)
	case protocol.OpDateTimeNTP:
		return r.dateNTP(op)
	case protocol.OpDateTimeTime:
		return r.dateTime(op)
	case protocol.OpLocaleLang:
		return r.localeLang(op)
	case protocol.OpLocaleKeymap:
		return r.inputKeymap(op)
	case protocol.OpSessionLock:
		return r.sessionLock()
	case protocol.OpPowerSuspend:
		return r.powerAction("suspend")
	case protocol.OpPowerHibernate:
		return r.powerAction("hibernate")
	case protocol.OpPowerBrightness:
		return r.powerBrightness(op)
	case protocol.OpPowerLid:
		return r.powerLid(op)
	default:
		return fmt.Errorf("refused: unknown op %q", op.Type)
	}
}

func (r *Runner) hyprctlBin() string {
	if r.Hyprctl != "" {
		return r.Hyprctl
	}
	return "hyprctl"
}

func (r *Runner) runHypr(args ...string) (string, error) {
	return r.runWith(true, true, r.hyprctlBin(), args...)
}

func (r *Runner) runSession(name string, args ...string) (string, error) {
	return r.runWith(true, false, name, args...)
}

func (r *Runner) runHost(name string, args ...string) (string, error) {
	return r.runWith(false, false, name, args...)
}

func (r *Runner) runWith(session, needHypr bool, name string, args ...string) (string, error) {
	if r.Run != nil {
		return r.Run(name, args...)
	}
	d := r.Timeout
	if d <= 0 {
		d = 4 * time.Second
	}
	if name == "bluetoothctl" && d < 10*time.Second {
		for _, a := range args {
			if a == "pair" || a == "scan" || a == "connect" {
				d = 10 * time.Second
				break
			}
		}
	}
	if !session {
		return runcmd.Run(d, name, args...)
	}
	var sess hyprsession.Session
	var err error
	if r.Discover != nil {
		sess, err = r.Discover()
	} else if needHypr {
		sess, err = hyprsession.Discover()
	} else {
		sess, err = hyprsession.DiscoverRuntime()
	}
	if err != nil {
		return "", fmt.Errorf("session: %w", err)
	}
	cmd := sess.Command(name, args...)
	out, err := runcmd.Prepared(d, cmd)
	if err != nil {
		return out, fmt.Errorf("%w [%s]", err, sess.String())
	}
	return out, nil
}

func (r *Runner) writeFile(path string, data []byte, perm os.FileMode) error {
	if r.WriteFile != nil {
		return r.WriteFile(path, data, perm)
	}
	// Command-spy tests set Run but not WriteFile — never touch the host.
	if r.Run != nil {
		return nil
	}
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func parentDir(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return "."
	}
	return path[:i]
}

func luaString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func luaBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func checkHyprOK(out string, err error) error {
	if err != nil {
		return fmt.Errorf("hyprctl eval: %w (%s)", err, strings.TrimSpace(out))
	}
	first := strings.TrimSpace(out)
	if i := strings.IndexByte(first, '\n'); i >= 0 {
		first = first[:i]
	}
	if first != "" && first != "ok" {
		return fmt.Errorf("hyprctl eval: %s", strings.TrimSpace(out))
	}
	return nil
}

// NeedHyprctl reports whether PATH has hyprctl (for diagnostics).
func NeedHyprctl() bool {
	_, err := exec.LookPath("hyprctl")
	return err == nil
}
