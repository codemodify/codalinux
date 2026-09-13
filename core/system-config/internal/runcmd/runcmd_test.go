package runcmd

import (
	"strings"
	"testing"
	"time"
)

func TestTimeout(t *testing.T) {
	_, err := Run(80*time.Millisecond, "sleep", "2")
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("got %v", err)
	}
}

func TestRunOK(t *testing.T) {
	out, err := Run(2*time.Second, "true")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
}
