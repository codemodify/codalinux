package rootguard

import (
	"os"
	"testing"
)

func TestAllowedNonRoot(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test host is uid 0")
	}
	if !Allowed() {
		t.Fatal("non-root must be allowed")
	}
	t.Setenv("CODA_SYSTEM_CONFIG_ALLOW_ROOT", "1")
	if !Allowed() {
		t.Fatal("allow-root env must not reject a non-root process")
	}
}
