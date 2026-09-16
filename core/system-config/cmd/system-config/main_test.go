package main

import "testing"

func TestParsePathFollow(t *testing.T) {
	path, follow := parsePathFollow([]string{"display"})
	if path != "display" || follow {
		t.Fatalf("got %q %v", path, follow)
	}
	path, follow = parsePathFollow([]string{"display", "--follow"})
	if path != "display" || !follow {
		t.Fatalf("got %q %v", path, follow)
	}
	path, follow = parsePathFollow([]string{"-f", "network"})
	if path != "network" || !follow {
		t.Fatalf("got %q %v", path, follow)
	}
	path, follow = parsePathFollow(nil)
	if path != "" || follow {
		t.Fatalf("empty args %q %v", path, follow)
	}
}
