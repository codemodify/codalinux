package applyexec

import (
	"fmt"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func (r *Runner) hyprMonitor(op protocol.PlanOp) error {
	if op.Output == "" {
		return fmt.Errorf("output required")
	}
	scale := op.Scale
	if scale <= 0 {
		scale = 1
	}
	mode := op.Mode
	if mode == "" {
		mode = "preferred"
	}
	expr := fmt.Sprintf("hl.monitor({ output = %s, mode = %s, position = %s, scale = %g })",
		luaString(op.Output), luaString(mode), luaString("auto"), scale)
	out, err := r.runHypr("eval", expr)
	if e := checkHyprOK(out, err); e != nil {
		return e
	}
	r.persistHypr(op)
	return nil
}
