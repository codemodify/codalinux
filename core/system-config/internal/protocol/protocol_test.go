package protocol

import (
	"strings"
	"testing"
)

func TestNormalizeAndKnownPath(t *testing.T) {
	if NormalizePath(" /Display/ ") != PathDisplay {
		t.Fatalf("normalize: %q", NormalizePath(" /Display/ "))
	}
	if !KnownPath("display") || !KnownPath("devices.pci") || !KnownPath("") {
		t.Fatal("expected starter paths")
	}
	if !KnownPath("network") || !KnownPath("audio") || !KnownPath("bluetooth") {
		t.Fatal("network/audio/bluetooth must be known")
	}
	if !KnownPath("input") || !KnownPath("datetime") || !KnownPath("locale") {
		t.Fatal("input/datetime/locale must be known")
	}
	if !KnownPath("printers") || !KnownPath("users") || !KnownPath("storage") {
		t.Fatal("printers/users/storage must be known")
	}
	if Settable("printers") || Settable("users") || Settable("storage") {
		t.Fatal("printers/users/storage are observe-only")
	}
	if !AllowedOp(OpDisplayPosition) || !AllowedOp(OpNetAirplane) {
		t.Fatal("new apply ops")
	}
	if AllowedOp("power.reboot") || AllowedOp("power.poweroff") {
		t.Fatal("reboot/poweroff must not be allowlisted")
	}
	if !Settable("network") || Settable("devices.pci") || Settable("hardware.dmi") {
		t.Fatal("settable allowlist")
	}
	if !AllowedOp(OpNetWiFiConnect) || AllowedOp("shell") {
		t.Fatal("apply allowlist")
	}
	log := ApplyOpsLog()
	if !strings.Contains(log, OpDisplayScale) || !strings.Contains(log, OpBTPair) || !strings.Contains(log, OpPowerBrightness) {
		t.Fatalf("allowlist log %q", log)
	}
	if len(ApplyOps) < 20 {
		t.Fatalf("allowlist too small: %d", len(ApplyOps))
	}
}

func TestEncodeDecode(t *testing.T) {
	line, err := Encode(Request{ID: "1", Op: OpGet, Path: "display"})
	if err != nil {
		t.Fatal(err)
	}
	req, err := DecodeRequest(line)
	if err != nil {
		t.Fatal(err)
	}
	if req.Op != OpGet || req.Path != PathDisplay || req.Version != Version {
		t.Fatalf("%+v", req)
	}
}
