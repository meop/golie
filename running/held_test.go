package running

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// a file nothing runs from is not held, and asking leaves it where it was
func TestHeldNotForAFileNothingRunsFrom(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if Held(path) {
		t.Error("held, though nothing runs from it")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("asking took the file: %v", err)
	}
}

// the test binary runs from its file: windows holds it, set aside under another name as a stamp sets a shim aside,
// and nothing else does
func TestHeldTheRunningProgramsFile(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	aside := filepath.Join(filepath.Dir(self), "."+filepath.Base(self)+".1.old")
	if err := os.Rename(self, aside); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Rename(aside, self) })
	if got, want := Held(aside), runtime.GOOS == "windows"; got != want {
		t.Errorf("Held(the running program, set aside) = %v, want %v", got, want)
	}
	if _, err := os.Stat(aside); err != nil {
		t.Fatalf("asking took the file: %v", err)
	}
}
