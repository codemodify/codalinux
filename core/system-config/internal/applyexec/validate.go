package applyexec

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	reIface     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,15}$`)
	reToken     = regexp.MustCompile(`^[A-Za-z0-9_.:@+/-]{1,64}$`)
	reLang      = regexp.MustCompile(`^[A-Za-z]{2}(_[A-Za-z]{2})?(\.[A-Za-z0-9_-]+)?(@[A-Za-z0-9_-]+)?$`)
	reTZ        = regexp.MustCompile(`^[A-Za-z0-9/_+-]{1,64}$`)
	reKeymap    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
	reBT        = regexp.MustCompile(`^(?i)([0-9A-F]{2}:){5}[0-9A-F]{2}$`)
	reNodeID    = regexp.MustCompile(`^[0-9]{1,8}$`)
	reBacklight = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	reLid       = regexp.MustCompile(`^(ignore|suspend|lock|poweroff|hibernate)$`)
)

func checkIface(s string) error {
	if !reIface.MatchString(s) {
		return fmt.Errorf("invalid interface %q", s)
	}
	return nil
}

func checkToken(label, s string) error {
	if !reToken.MatchString(s) {
		return fmt.Errorf("invalid %s %q", label, s)
	}
	return nil
}

func checkSSID(s string) error {
	if s == "" || len(s) > 32 {
		return fmt.Errorf("invalid ssid length")
	}
	for _, r := range s {
		if r == 0 || r == '\n' || r == '\r' {
			return fmt.Errorf("invalid ssid")
		}
	}
	return nil
}

func checkPSK(s string) error {
	if s == "" {
		return nil
	}
	if len(s) < 8 || len(s) > 63 {
		return fmt.Errorf("invalid psk length")
	}
	for _, r := range s {
		if r == 0 || r == '\n' {
			return fmt.Errorf("invalid psk")
		}
	}
	return nil
}

func checkMethod(s string) error {
	if s != "dhcp" && s != "static" {
		return fmt.Errorf("method must be dhcp or static")
	}
	return nil
}

func checkCIDR(s string) error {
	if s == "" {
		return nil
	}
	if _, _, err := net.ParseCIDR(s); err != nil {
		if ip := net.ParseIP(s); ip == nil {
			return fmt.Errorf("invalid address %q", s)
		}
	}
	return nil
}

func checkIP(s string) error {
	if s == "" {
		return nil
	}
	if net.ParseIP(s) == nil {
		return fmt.Errorf("invalid ip %q", s)
	}
	return nil
}

func checkBT(s string) error {
	if !reBT.MatchString(s) {
		return fmt.Errorf("invalid bluetooth address")
	}
	return nil
}

func checkLang(s string) error {
	if s != "C" && s != "C.UTF-8" && s != "POSIX" && !reLang.MatchString(s) {
		return fmt.Errorf("invalid lang %q", s)
	}
	return nil
}

func checkTZ(s string) error {
	if s != "UTC" && !reTZ.MatchString(s) {
		return fmt.Errorf("invalid timezone %q", s)
	}
	if strings.Contains(s, "..") {
		return fmt.Errorf("invalid timezone")
	}
	return nil
}

func checkKeymap(s string) error {
	if !reKeymap.MatchString(s) {
		return fmt.Errorf("invalid keymap %q", s)
	}
	return nil
}

func checkNodeID(s string) error {
	if s == "" || s == "@DEFAULT_AUDIO_SINK@" || s == "@DEFAULT_AUDIO_SOURCE@" {
		return nil
	}
	if reNodeID.MatchString(s) {
		return nil
	}
	// allow a PipeWire node name (no spaces / shell)
	if len(s) <= 96 {
		for _, r := range s {
			if r > unicode.MaxASCII || r == 0 || r == ' ' || r == '\n' || r == ';' || r == '|' {
				return fmt.Errorf("invalid node id")
			}
		}
		return nil
	}
	return fmt.Errorf("invalid node id")
}

func checkTime(s string) error {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05 MST",
	} {
		if _, err := time.Parse(layout, s); err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid time %q", s)
}

func onOff(enabled *bool) string {
	if enabled != nil && *enabled {
		return "on"
	}
	return "off"
}

func yesNo(enabled *bool) string {
	if enabled != nil && *enabled {
		return "yes"
	}
	return "no"
}

func trueFalse(enabled *bool) string {
	if enabled != nil && *enabled {
		return "true"
	}
	return "false"
}
