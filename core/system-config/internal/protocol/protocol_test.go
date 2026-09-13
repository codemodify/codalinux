package protocol

import "testing"

func TestNormalizeAndKnownPath(t *testing.T) {
	if NormalizePath(" /Display/ ") != PathDisplay {
		t.Fatalf("normalize: %q", NormalizePath(" /Display/ "))
	}
	if !KnownPath("display") || !KnownPath("devices.pci") || !KnownPath("") {
		t.Fatal("expected starter paths")
	}
	if KnownPath("network") {
		t.Fatal("network is not a v1 starter path")
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
