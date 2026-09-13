package plan

import "github.com/codemodify/codalinux/core/system-config/internal/protocol"

func FromInput(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.InputModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	var ops []protocol.PlanOp
	if want.KBLayout != "" && want.KBLayout != have.KBLayout {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpInputKBLayout, Value: want.KBLayout})
	}
	if want.Keymap != "" && want.Keymap != have.Keymap {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpInputKeymap, Value: want.Keymap})
	}
	if jsonHas(desired, "pointer_speed") && abs(want.PointerSpeed-have.PointerSpeed) > 0.01 {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpInputPointerSpeed, Speed: want.PointerSpeed})
	}
	if jsonHas(desired, "natural_scroll") && want.NaturalScroll != have.NaturalScroll {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpInputNaturalScroll, Enabled: boolPtr(want.NaturalScroll)})
	}
	if jsonHas(desired, "tap_to_click") && want.TapToClick != have.TapToClick {
		ops = append(ops, protocol.PlanOp{Type: protocol.OpInputTapToClick, Enabled: boolPtr(want.TapToClick)})
	}
	return protocol.Plan{Path: protocol.PathInput, Ops: ops}, nil
}
