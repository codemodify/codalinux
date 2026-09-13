package plan

import (
	"fmt"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func FromDisplay(desired, observed []byte) (protocol.Plan, error) {
	var want, have protocol.DisplayModel
	if err := unmarshal(desired, observed, &want, &have); err != nil {
		return protocol.Plan{}, err
	}
	byName := map[string]protocol.Output{}
	for _, o := range have.Outputs {
		byName[o.Name] = o
	}
	var ops []protocol.PlanOp
	for _, w := range want.Outputs {
		if w.Name == "" {
			continue
		}
		h := byName[w.Name]
		mode := w.Mode
		if mode == "" && h.Width > 0 && h.Height > 0 {
			hz := h.RefreshHz
			if hz == 0 {
				hz = 60
			}
			mode = fmt.Sprintf("%dx%d@%d", h.Width, h.Height, hz)
		}
		if w.Scale > 0 && (h.Scale == 0 || abs(h.Scale-w.Scale) > 0.01) {
			ops = append(ops, protocol.PlanOp{
				Type: protocol.OpDisplayScale, Output: w.Name, Scale: w.Scale, Mode: mode,
			})
		}
		if w.Mode != "" && w.Mode != h.Mode && w.Mode != modeFrom(h) {
			ops = append(ops, protocol.PlanOp{
				Type: protocol.OpDisplayMode, Output: w.Name, Mode: w.Mode, Scale: orScale(w.Scale, h.Scale),
			})
		}
	}
	return protocol.Plan{Path: protocol.PathDisplay, Ops: ops}, nil
}

func modeFrom(o protocol.Output) string {
	if o.Mode != "" {
		return o.Mode
	}
	if o.Width == 0 || o.Height == 0 {
		return ""
	}
	hz := o.RefreshHz
	if hz == 0 {
		hz = 60
	}
	return fmt.Sprintf("%dx%d@%d", o.Width, o.Height, hz)
}

func orScale(a, b float64) float64 {
	if a > 0 {
		return a
	}
	if b > 0 {
		return b
	}
	return 1
}
