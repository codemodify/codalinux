package applyexec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

type hyprState struct {
	Output   string  `json:"output,omitempty"`
	Mode     string  `json:"mode,omitempty"`
	Scale    float64 `json:"scale,omitempty"`
	KBLayout string  `json:"kb_layout,omitempty"`
	Speed    float64 `json:"pointer_speed"`
	Natural  *bool   `json:"natural_scroll,omitempty"`
	Tap      *bool   `json:"tap_to_click,omitempty"`
}

func (r *Runner) persistHypr(op protocol.PlanOp) {
	if r.Run != nil && r.WriteFile == nil {
		return
	}
	home := ""
	if r.Discover != nil {
		if s, err := r.Discover(); err == nil {
			home = s.Home
		}
	} else if s, err := hyprsession.DiscoverRuntime(); err == nil {
		home = s.Home
	}
	if home == "" {
		if u := os.Getenv("HOME"); u != "" && os.Getuid() != 0 {
			home = u
		}
	}
	if home == "" {
		return
	}
	dir := filepath.Join(home, ".config", "hypr")
	statePath := filepath.Join(dir, "coda-system-config.state.json")
	luaPath := filepath.Join(dir, "coda-system-config.lua")
	st := hyprState{}
	if b, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	switch op.Type {
	case protocol.OpDisplayScale, protocol.OpDisplayMode:
		st.Output = op.Output
		if op.Mode != "" {
			st.Mode = op.Mode
		}
		if op.Scale > 0 {
			st.Scale = op.Scale
		}
	case protocol.OpInputKBLayout:
		st.KBLayout = op.Value
	case protocol.OpInputPointerSpeed:
		st.Speed = op.Speed
	case protocol.OpInputNaturalScroll:
		st.Natural = op.Enabled
	case protocol.OpInputTapToClick:
		st.Tap = op.Enabled
	}
	raw, _ := json.MarshalIndent(st, "", "  ")
	_ = r.writeFile(statePath, raw, 0o644)
	_ = r.writeFile(luaPath, []byte(st.lua()), 0o644)
}

func (s hyprState) lua() string {
	out := "-- written by system-config-apply; loaded from hyprland.lua\n"
	if s.Output != "" && (s.Scale > 0 || s.Mode != "") {
		mode := s.Mode
		if mode == "" {
			mode = "preferred"
		}
		scale := s.Scale
		if scale <= 0 {
			scale = 1
		}
		out += fmt.Sprintf("hl.monitor({ output = %s, mode = %s, position = \"auto\", scale = %g })\n",
			luaString(s.Output), luaString(mode), scale)
	}
	var parts []string
	if s.KBLayout != "" {
		parts = append(parts, "kb_layout = "+luaString(s.KBLayout))
	}
	if s.Speed != 0 {
		parts = append(parts, fmt.Sprintf("sensitivity = %g", s.Speed))
	}
	touch := ""
	if s.Natural != nil {
		touch += "natural_scroll = " + luaBool(*s.Natural)
	}
	if s.Tap != nil {
		if touch != "" {
			touch += ", "
		}
		touch += "tap_to_click = " + luaBool(*s.Tap)
	}
	if touch != "" {
		parts = append(parts, "touchpad = { "+touch+" }")
	}
	if len(parts) > 0 {
		out += "hl.input({\n"
		for _, p := range parts {
			out += "    " + p + ",\n"
		}
		out += "})\n"
	}
	return out
}
