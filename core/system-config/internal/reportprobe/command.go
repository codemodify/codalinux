package reportprobe

import "os/exec"

func execCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
