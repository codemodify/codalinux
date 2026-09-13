package applyexec

import (
	"strings"
	"testing"

	"github.com/codemodify/codalinux/core/system-config/internal/protocol"
)

func TestRefuseUnknown(t *testing.T) {
	r := New()
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{Type: "shell", Mode: "rm -rf /"}}})
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("got %v", err)
	}
}

func TestDisplayScaleEval(t *testing.T) {
	var got []string
	r := New()
	r.Run = func(name string, args ...string) (string, error) {
		got = append([]string{name}, args...)
		return "ok\n", nil
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpDisplayScale, Output: "Virtual-1", Scale: 2, Mode: "1920x1080@60",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[1] != "eval" {
		t.Fatalf("args %v", got)
	}
	if !strings.Contains(got[2], `hl.monitor({`) || !strings.Contains(got[2], `scale = 2`) {
		t.Fatalf("expr %s", got[2])
	}
	if strings.Contains(got[2], "keyword") {
		t.Fatal("must not use keyword")
	}
}

func TestHyprctlErrorPropagates(t *testing.T) {
	r := New()
	r.Run = func(string, ...string) (string, error) {
		return "keyword can't work with non-legacy parsers. Use eval.", nil
	}
	err := r.Exec(protocol.Plan{Ops: []protocol.PlanOp{{
		Type: protocol.OpDisplayScale, Output: "eDP-1", Scale: 2,
	}}})
	if err == nil {
		t.Fatal("expected error")
	}
}
