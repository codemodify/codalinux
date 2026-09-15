package applyexec

import (
	"fmt"
	"os"
	"os/user"
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
	matchKey, matchVal := r.audioPersistMatch(id)
	if matchVal == "" {
		matchVal = id
	}
	if !okPersistLabel(matchVal) {
		return
	}
	home := r.sessionHome()
	if home == "" {
		return
	}
	dir := filepath.Join(home, ".config", "wireplumber", "wireplumber.conf.d")
	path := filepath.Join(dir, "51-coda-defaults.conf")
	which := "sink"
	if kind == protocol.OpAudioDefaultSource {
		which = "source"
	}
	sink, source, sinkKey, sourceKey := "", "", "node.name", "node.name"
	if b, err := os.ReadFile(path); err == nil {
		sink, source, sinkKey, sourceKey = parseCodaAudioConf(string(b))
	}
	if which == "sink" {
		sink, sinkKey = matchVal, matchKey
	} else {
		source, sourceKey = matchVal, matchKey
	}
	body := "# written by system-config-apply; WirePlumber 0.5 session defaults\n"
	body += writeAudioRule(sink, source, sinkKey, sourceKey)
	_ = r.writeFile(path, []byte(body), 0o644)
}

func (r *Runner) audioPersistMatch(id string) (key, value string) {
	raw, err := r.runSession("wpctl", "inspect", id)
	if err != nil {
		return "", id
	}
	name, nick, desc := parseWpInspectProps(raw)
	if okPersistName(name) {
		return "node.name", name
	}
	if okPersistName(nick) {
		return "node.nick", nick
	}
	if okPersistLabel(desc) {
		return "node.description", desc
	}
	if okPersistLabel(name) {
		return "node.name", name
	}
	if okPersistLabel(nick) {
		return "node.nick", nick
	}
	return "", id
}

func parseWpInspectProps(raw string) (name, nick, desc string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(k), "*"))
		v = strings.Trim(strings.TrimSpace(v), `"`)
		if v == "" || strings.EqualFold(v, "(null)") {
			continue
		}
		switch {
		case strings.HasSuffix(k, "node.name") && !strings.Contains(k, "description"):
			name = v
		case strings.HasSuffix(k, "node.nick"):
			nick = v
		case strings.HasSuffix(k, "node.description"):
			desc = v
		}
	}
	return name, nick, desc
}

func okPersistName(s string) bool {
	if s == "" || strings.ContainsAny(s, " \t") {
		return false
	}
	return okPersistLabel(s)
}

func okPersistLabel(s string) bool {
	if s == "" || len(s) > 96 {
		return false
	}
	for _, r := range s {
		if r == 0 || r == '\n' || r == '\r' || r == ';' || r == '|' || r == '"' {
			return false
		}
	}
	return true
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
	if uid := os.Getenv("CODA_SYSTEM_CONFIG_UID"); uid != "" && uid != "0" {
		if u, err := user.LookupId(uid); err == nil && u.HomeDir != "" {
			return u.HomeDir
		}
	}
	return ""
}

func parseCodaAudioConf(s string) (sink, source, sinkKey, sourceKey string) {
	sinkKey, sourceKey = "node.name", "node.name"
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "# coda-sink-match="):
			sinkKey = strings.TrimPrefix(line, "# coda-sink-match=")
		case strings.HasPrefix(line, "# coda-source-match="):
			sourceKey = strings.TrimPrefix(line, "# coda-source-match=")
		case strings.HasPrefix(line, "# coda-sink="):
			sink = strings.TrimPrefix(line, "# coda-sink=")
		case strings.HasPrefix(line, "# coda-source="):
			source = strings.TrimPrefix(line, "# coda-source=")
		}
	}
	return sink, source, sinkKey, sourceKey
}

func writeAudioRule(sink, source, sinkKey, sourceKey string) string {
	var b strings.Builder
	if sink != "" {
		b.WriteString("# coda-sink=")
		b.WriteString(sink)
		b.WriteString("\n")
		if sinkKey != "" {
			b.WriteString("# coda-sink-match=")
			b.WriteString(sinkKey)
			b.WriteString("\n")
		}
	}
	if source != "" {
		b.WriteString("# coda-source=")
		b.WriteString(source)
		b.WriteString("\n")
		if sourceKey != "" {
			b.WriteString("# coda-source-match=")
			b.WriteString(sourceKey)
			b.WriteString("\n")
		}
	}
	var rules []string
	add := func(key, name string) {
		if name == "" {
			return
		}
		if key == "" {
			// Numeric wpctl id only — still write markers so e2e can see the file.
			return
		}
		if key != "node.name" && key != "node.nick" && key != "node.description" {
			key = "node.name"
		}
		rules = append(rules, "  {\n    matches = [\n      { "+key+" = "+luaString(name)+" }\n    ]\n    actions = {\n      update-props = {\n        priority.session = 2000\n        priority.driver = 2000\n      }\n    }\n  }")
	}
	add(sinkKey, sink)
	add(sourceKey, source)
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
