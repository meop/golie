package shim

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func stamper(t *testing.T) Stamper {
	t.Helper()
	root := t.TempDir()
	return Stamper{Tool: "tool", BinDir: filepath.Join(root, "bin"), MalletDir: filepath.Join(root, "vendor"), Owned: filepath.Join(root, "extract")}
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

// stamped is a template's bytes with mallet's trailer: the path alone, or the
// path and its arguments.
func stamped(target string, args ...string) string {
	payload, magic := target, "SHIM"
	if len(args) > 0 {
		payload, magic = strings.Join(append([]string{target}, args...), "\x00"), "SHM2"
	}
	n := make([]byte, 4)
	binary.LittleEndian.PutUint32(n, uint32(len(payload)))
	return "a template that names " + target + payload + string(n) + magic
}

func TestIsShimReadsTheStampedTarget(t *testing.T) {
	s := stamper(t)
	ours := s.Path("ours")
	write(t, ours, stamped(filepath.Join(s.Owned, "pkg", "bin", "ours")))
	withArgs := s.Path("with-args")
	write(t, withArgs, stamped(filepath.Join(s.Owned, "pwsh", "pwsh.exe"), "-nol"))
	theirs := s.Path("theirs")
	write(t, theirs, stamped("/somewhere/else/theirs"))
	sibling := s.Path("sibling")
	write(t, sibling, stamped(s.Owned+"-other"+string(filepath.Separator)+"x"))
	plain := s.Path("plain")
	write(t, plain, "a binary that only mentions "+filepath.Join(s.Owned, "pkg"))

	if !s.IsShim(ours) || !s.IsShim(withArgs) {
		t.Error("a wedge shim stamped with a target under Owned is not recognised")
	}
	if s.IsShim(theirs) || s.IsShim(sibling) || s.IsShim(plain) || s.IsShim(s.Path("absent")) {
		t.Error("something the tool did not stamp was taken for its shim")
	}
	if target, ok := StampedTarget(withArgs); !ok || target != filepath.Join(s.Owned, "pwsh", "pwsh.exe") {
		t.Errorf("StampedTarget = %q, %v", target, ok)
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
