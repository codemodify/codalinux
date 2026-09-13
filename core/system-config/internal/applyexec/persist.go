package applyexec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codemodify/codalinux/core/system-config/internal/hyprsession"
	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

type hyprOutput struct {
	Output   string  `json:"output"`
	Mode     string  `json:"mode,omitempty"`
	Scale    float64 `json:"scale,omitempty"`
	Position string  `json:"position,omitempty"`
}

type hyprState struct {
	Outputs  []hyprOutput `json:"outputs,omitempty"`
	Output   string       `json:"output,omitempty"`
	Mode     string       `json:"mode,omitempty"`
	Scale    float64      `json:"scale,omitempty"`
	Position string       `json:"position,omitempty"`
	KBLayout string       `json:"kb_layout,omitempty"`
	Speed    float64      `json:"pointer_speed"`
	Natural  *bool        `json:"natural_scroll,omitempty"`
	Tap      *bool        `json:"tap_to_click,omitempty"`
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
	st.migrate()
	switch op.Type {
	case protocol.OpDisplayScale, protocol.OpDisplayMode, protocol.OpDisplayPosition:
		st.upsertOutput(hyprOutput{
			Output:   op.Output,
			Mode:     op.Mode,
			Scale:    op.Scale,
			Position: op.Position,
		})
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

func (s *hyprState) migrate() {
	if len(s.Outputs) > 0 {
		return
	}
	if s.Output == "" {
		return
	}
	s.Outputs = []hyprOutput{{
		Output: s.Output, Mode: s.Mode, Scale: s.Scale, Position: s.Position,
	}}
}

func (s *hyprState) upsertOutput(o hyprOutput) {
	if o.Output == "" {
		return
	}
	for i := range s.Outputs {
		if s.Outputs[i].Output != o.Output {
			continue
		}
		if o.Mode != "" {
			s.Outputs[i].Mode = o.Mode
		}
		if o.Scale > 0 {
			s.Outputs[i].Scale = o.Scale
		}
		if o.Position != "" {
			s.Outputs[i].Position = o.Position
		}
		s.syncLegacy()
		return
	}
	s.Outputs = append(s.Outputs, o)
	s.syncLegacy()
}

func (s *hyprState) syncLegacy() {
	if len(s.Outputs) == 0 {
		return
	}
	cur := s.Outputs[len(s.Outputs)-1]
	s.Output, s.Mode, s.Scale, s.Position = cur.Output, cur.Mode, cur.Scale, cur.Position
}

func (s hyprState) lua() string {
	out := "-- written by system-config-apply; loaded from hyprland.lua\n"
	outs := s.Outputs
	if len(outs) == 0 && s.Output != "" {
		outs = []hyprOutput{{Output: s.Output, Mode: s.Mode, Scale: s.Scale, Position: s.Position}}
	}
	for _, o := range outs {
		if o.Output == "" || (o.Scale <= 0 && o.Mode == "" && o.Position == "") {
			continue
		}
		mode := o.Mode
		if mode == "" {
			mode = "preferred"
		}
		scale := o.Scale
		if scale <= 0 {
			scale = 1
		}
		pos := o.Position
		if pos == "" {
			pos = "auto"
		}
		out += fmt.Sprintf("hl.monitor({ output = %s, mode = %s, position = %s, scale = %g })\n",
			luaString(o.Output), luaString(mode), luaString(pos), scale)
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
