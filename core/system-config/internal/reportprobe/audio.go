package reportprobe

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
	"github.com/codemodify/codalinux/core/system-config/internal/rpc"
)

var wpLine = regexp.MustCompile(`(\*)?\s*(\d+)\.\s+(.+?)(?:\s+\[vol:\s*([0-9.]+)(?:\s+MUTED)?\])?\s*$`)

func (p *Probe) audio() (json.RawMessage, error) {
	m := protocol.AudioModel{}
	raw, err := p.cmdSession("wpctl", "status")
	if err != nil {
		return rpc.Raw(m), nil
	}
	section := ""
	for _, line := range strings.Split(raw, "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "Sinks:"):
			section = "sinks"
			continue
		case strings.HasPrefix(trim, "Sources:"):
			section = "sources"
			continue
		case strings.HasPrefix(trim, "Filters:") || strings.HasPrefix(trim, "Streams:") || strings.HasPrefix(trim, "Video"):
			section = ""
			continue
		}
		sub := wpLine.FindStringSubmatch(line)
		if sub == nil || (section != "sinks" && section != "sources") {
			continue
		}
		vol, _ := strconv.ParseFloat(sub[4], 64)
		n := protocol.AudioNode{
			ID:      sub[2],
			Name:    strings.TrimSpace(sub[3]),
			Volume:  vol,
			Mute:    sub[5] != "",
			Default: sub[1] == "*",
		}
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
	if vol, err := p.cmdSession("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@"); err == nil {
		m.Volume, m.Mute = parseWpVolume(vol)
	}
	return rpc.Raw(m), nil
}

func parseWpVolume(s string) (float64, *bool) {
	// Volume: 0.50 or Volume: 0.50 [MUTED]
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
