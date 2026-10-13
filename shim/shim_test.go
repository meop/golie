package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func stamper(t *testing.T) Stamper {
	t.Helper()
	root := t.TempDir()
	return Stamper{Tool: "tool", BinDir: filepath.Join(root, "bin"), KebabDir: filepath.Join(root, "vendor"), Owned: filepath.Join(root, "extract")}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestIsShimWantsTheMarkerAndAnOwnedTarget(t *testing.T) {
	s := stamper(t)
	ours := s.Path("ours")
	write(t, ours, "…"+sheeshMarker+"…"+filepath.Join(s.Owned, "pkg", "bin", "ours"))
	theirs := s.Path("theirs")
	write(t, theirs, "…"+sheeshMarker+"…/somewhere/else/theirs")
	plain := s.Path("plain")
	write(t, plain, "a binary that only mentions "+s.Owned)

	if !s.IsShim(ours) {
		t.Error("a sheesh shim pointing into Owned is not recognised")
	}
	if s.IsShim(theirs) || s.IsShim(plain) || s.IsShim(s.Path("absent")) {
		t.Error("something the tool did not stamp was taken for its shim")
	}
}

func TestExeName(t *testing.T) {
	want := "go1.26"
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if got := ExeName("go1.26"); got != want {
		t.Errorf("ExeName = %s, want %s", got, want)
	}
}

func TestCreateAndRemoveLeaveAForeignFileAlone(t *testing.T) {
	s := stamper(t)
	foreign := s.Path("tool")
	write(t, foreign, "somebody else's binary")

	err := s.Create("tool", filepath.Join(s.Owned, "tool"), false)
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("Create over a foreign file returned %v", err)
	}
	if err := s.Remove("tool"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(foreign); string(got) != "somebody else's binary" {
		t.Error("a file the tool did not stamp was removed or replaced")
	}
	if err := s.Remove("absent"); err != nil {
		t.Errorf("removing a shim that is not there returned %v", err)
	}
}
