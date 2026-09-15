//go:build linux

package hyprsession

import "testing"

func TestSessionCredsRootDropsToLive(t *testing.T) {
	s := Session{UID: 1000, GID: 1000, Groups: []uint32{1000, 14}}
	c := sessionCreds(0, s)
	if c == nil || c.Uid != 1000 || c.Gid != 1000 {
		t.Fatalf("%+v", c)
	}
}

func TestSessionCredsNoDropWhenAlreadyUser(t *testing.T) {
	s := Session{UID: 1000, GID: 1000}
	if c := sessionCreds(1000, s); c != nil {
		t.Fatalf("unexpected %+v", c)
	}
	if c := sessionCreds(0, Session{UID: 0}); c != nil {
		t.Fatalf("root session %+v", c)
	}
}
