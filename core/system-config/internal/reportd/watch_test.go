package reportd

import (
	"context"
	"testing"
	"time"
)

func TestWatchReturnsOnCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() {
		Watch(ctx, "/tmp/coda-no-system-configd.sock", []string{"display"}, 50*time.Millisecond, nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Watch did not return after context cancel")
	}
}
