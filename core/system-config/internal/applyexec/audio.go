package applyexec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) audioID(op protocol.PlanOp) (string, error) {
	id := op.ID
	if id == "" {
		id = op.Name
	}
	if id == "" {
		id = "@DEFAULT_AUDIO_SINK@"
	}
	if err := checkNodeID(id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *Runner) audioDefault(op protocol.PlanOp) error {
	id, err := r.audioID(op)
	if err != nil {
		return err
	}
	out, err := r.runSession("wpctl", "set-default", id)
	if err != nil {
		return fmt.Errorf("wpctl set-default: %w (%s)", err, strings.TrimSpace(out))
	}
	r.persistAudioDefault(op.Type, id)
	return nil
}

func (r *Runner) persistAudioDefault(kind, id string) {
	if r.Run != nil && r.WriteFile == nil {
		return
	}
	name := r.audioNodeName(id)
	if name == "" {
		name = id
	}
	if err := checkNodeID(name); err != nil {
		return
	}
	home := r.sessionHome()
	if home == "" {
		return
	}
	dir := filepath.Join(home, ".config", "wireplumber", "wireplumber.conf.d")
	path := filepath.Join(dir, "51-coda-defaults.conf")
	key := "playback"
	if kind == protocol.OpAudioDefaultSource {
		key = "capture"
	}
	// Read existing so we keep the other default.
	sink, source := "", ""
	if b, err := os.ReadFile(path); err == nil {
		sink, source = parseCodaAudioConf(string(b))
	}
	if key == "playback" {
		sink = name
	} else {
		source = name
	}
	body := "# written by system-config-apply; WirePlumber 0.5 session defaults\n"
	body += writeAudioRule(sink, source)
	_ = r.writeFile(path, []byte(body), 0o644)
}

func (r *Runner) audioNodeName(id string) string {
	raw, err := r.runSession("wpctl", "inspect", id)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "node.name") {
			if _, v, ok := strings.Cut(line, "="); ok {
				return strings.Trim(strings.TrimSpace(v), `"`)
			}
		}
	}
	return ""
}

func (r *Runner) sessionHome() string {
	if r.Discover != nil {
		if s, err := r.Discover(); err == nil && s.Home != "" {
			return s.Home
		}
	}
	// Injected Run is a command spy — do not touch the host session.
	if r.Run != nil {
		return ""
	}
	if s, err := hyprsession.DiscoverRuntime(); err == nil && s.Home != "" {
		return s.Home
	}
	if u := os.Getenv("HOME"); u != "" && os.Getuid() != 0 {
		return u
	}
	return ""
}

func parseCodaAudioConf(s string) (sink, source string) {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# coda-sink=") {
			sink = strings.TrimPrefix(line, "# coda-sink=")
		}
		if strings.HasPrefix(line, "# coda-source=") {
			source = strings.TrimPrefix(line, "# coda-source=")
		}
	}
	return sink, source
}

func writeAudioRule(sink, source string) string {
	var b strings.Builder
	if sink != "" {
		b.WriteString("# coda-sink=")
		b.WriteString(sink)
		b.WriteString("\n")
	}
	if source != "" {
		b.WriteString("# coda-source=")
		b.WriteString(source)
		b.WriteString("\n")
	}
	var rules []string
	add := func(name string) {
		if name == "" {
			return
		}
		rules = append(rules, "  {\n    matches = [\n      { node.name = "+luaString(name)+" }\n    ]\n    actions = {\n      update-props = {\n        priority.session = 2000\n        priority.driver = 2000\n      }\n    }\n  }")
	}
	add(sink)
	add(source)
	if len(rules) > 0 {
		b.WriteString("monitor.alsa.rules = [\n")
		b.WriteString(strings.Join(rules, ",\n"))
		b.WriteString("\n]\n")
	}
	return b.String()
}

func (r *Runner) audioVolume(op protocol.PlanOp) error {
	id, err := r.audioID(op)
	if err != nil {
		return err
	}
	v := op.Volume
	if v < 0 {
		v = 0
	}
	if v > 1.5 {
		v = 1.5
	}
	arg := fmt.Sprintf("%g", v)
	out, err := r.runSession("wpctl", "set-volume", id, arg)
	if err != nil {
		return fmt.Errorf("wpctl set-volume: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

func (r *Runner) audioMute(op protocol.PlanOp) error {
	id, err := r.audioID(op)
	if err != nil {
		return err
	}
	flag := "0"
	if op.Mute != nil && *op.Mute {
		flag = "1"
	}
	out, err := r.runSession("wpctl", "set-mute", id, flag)
	if err != nil {
		return fmt.Errorf("wpctl set-mute: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}
