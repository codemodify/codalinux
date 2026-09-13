package reportprobe

import (
	"time"

	"github.com/codemodify/codalinux/core/system-config/internal/runcmd"
)

func execCommand(name string, args ...string) ([]byte, error) {
	out, err := runcmd.Run(2500*time.Millisecond, name, args...)
	return []byte(out), err
}
