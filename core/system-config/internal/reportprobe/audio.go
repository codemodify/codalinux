package reportprobe

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

var wpID = regexp.MustCompile(`(?m)^id\s+(\d+)`)

func (p *Probe) audio() (json.RawMessage, error) {
	m := protocol.AudioModel{}
	raw, err := p.cmdSession("wpctl", "status")
	if err == nil {
		parseWpStatus(raw, &m)
	}
	if id, name := p.inspectDefault("@DEFAULT_AUDIO_SINK@"); id != "" {
		m.DefaultSink = id
		if name != "" {
			found := false
			for i := range m.Sinks {
				if m.Sinks[i].ID == id {
					m.Sinks[i].Default = true
					found = true
				}
			}
			if !found {
				m.Sinks = append(m.Sinks, protocol.AudioNode{ID: id, Name: name, Default: true})
			}
		}
	}
	if id, name := p.inspectDefault("@DEFAULT_AUDIO_SOURCE@"); id != "" {
		m.DefaultSource = id
		if name != "" {
			found := false
			for i := range m.Sources {
				if m.Sources[i].ID == id {
					m.Sources[i].Default = true
					found = true
				}
			}
			if !found {
				m.Sources = append(m.Sources, protocol.AudioNode{ID: id, Name: name, Default: true})
			}
		}
	}
	if vol, err := p.cmdSession("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"); err == nil {
		m.Volume, m.Mute = parseWpVolume(vol)
		for i := range m.Sinks {
			if m.Sinks[i].Default || m.Sinks[i].ID == m.DefaultSink {
				m.Sinks[i].Volume = m.Volume
				if m.Mute != nil {
					m.Sinks[i].Mute = *m.Mute
				}
			}
		}
	}
	return rpc.Raw(m), nil
}

func parseWpStatus(raw string, m *protocol.AudioModel) {
	section := ""
	for _, line := range strings.Split(raw, "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.Contains(trim, "Sinks:"):
			section = "sinks"
			continue
		case strings.Contains(trim, "Sources:"):
			section = "sources"
			continue
		case strings.Contains(trim, "Filters:") || strings.Contains(trim, "Streams:") || strings.HasPrefix(trim, "Video"):
			section = ""
			continue
		}
		def, id, name, vol := parseWpNodeLine(line)
		if id == "" || (section != "sinks" && section != "sources") {
			continue
		}
		n := protocol.AudioNode{ID: id, Name: name, Volume: vol, Default: def}
		if section == "sinks" {
			m.Sinks = append(m.Sinks, n)
			if n.Default {
				m.DefaultSink = n.ID
			}
		} else {
			m.Sources = append(m.Sources, n)
			if n.Default {
				m.DefaultSource = n.ID
			}
		}
	}
}

func parseWpNodeLine(line string) (def bool, id, name string, vol float64) {
	// "│  *   52. Built-in Audio Analog Stereo [vol: 0.50]"
	star := strings.Contains(line, "*")
	re := regexp.MustCompile(`(\d+)\.\s+(.+?)(?:\s+\[vol:\s*([0-9.]+))?`)
	sub := re.FindStringSubmatch(line)
	if sub == nil {
		return false, "", "", 0
	}
	id = sub[1]
	name = strings.TrimSpace(sub[2])
	if i := strings.Index(name, "[vol:"); i >= 0 {
		name = strings.TrimSpace(name[:i])
	}
	if sub[3] != "" {
		vol, _ = strconv.ParseFloat(sub[3], 64)
	}
	return star, id, name, vol
}

func (p *Probe) inspectDefault(node string) (id, name string) {
	raw, err := p.cmdSession("wpctl", "inspect", node)
	if err != nil {
		return "", ""
	}
	if sub := wpID.FindStringSubmatch(raw); len(sub) > 1 {
		id = sub[1]
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "node.description") || strings.Contains(line, "node.nick") || strings.Contains(line, "node.name") {
			if _, v, ok := strings.Cut(line, "="); ok {
				name = strings.Trim(strings.TrimSpace(v), `"`)
				break
			}
		}
	}
	return id, name
}

func parseWpVolume(s string) (float64, *bool) {
	fields := strings.Fields(s)
	var v float64
	muted := false
	for i, f := range fields {
		if strings.EqualFold(f, "Volume:") && i+1 < len(fields) {
			v, _ = strconv.ParseFloat(fields[i+1], 64)
		}
		if strings.Contains(f, "MUTED") {
			muted = true
		}
	}
	return v, &muted
}
