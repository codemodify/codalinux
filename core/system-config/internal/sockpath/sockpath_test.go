package sockpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixupSocketMode(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "coda", "apply.sock")
	if err := EnsureDir(sock); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sock, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := FixupSocket(sock); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(sock)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o660 {
		t.Fatalf("mode %o want 0660", st.Mode().Perm())
	}
}

func TestEnsureDirMode(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "coda", "d.sock")
	if err := EnsureDir(sock); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Dir(sock))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o750 {
		t.Fatalf("dir mode %o want 0750", st.Mode().Perm())
	}
}
