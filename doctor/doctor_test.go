package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/meop/golie/ui"
	"github.com/meop/golie/vendored"
)

func capture(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	ui.SetOutput(&buf)
	ui.Reset()
	t.Cleanup(func() { ui.SetOutput(os.Stdout); ui.Reset() })
	return &buf
}

// TestVendoredSaysTheVersionAWholeReleaseHasOrWhatInstallsOne: a vendored
// tool reads as present, with its version in brackets as refresh says it,
// only when the dir holds every file of a release; else the line names the
// command that installs it.
func TestVendoredSaysTheVersionAWholeReleaseHasOrWhatInstallsOne(t *testing.T) {
	dir := t.TempDir()
	r := vendored.Release{Files: []string{"tool"}}
	buf := capture(t)

	Vendored("tool", r, dir, "x refresh")
	if err := os.WriteFile(filepath.Join(dir, vendored.VersionFile), []byte("1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	Vendored("tool", r, dir, "x refresh")
	if err := os.WriteFile(filepath.Join(dir, "tool"), []byte("bin"), 0755); err != nil {
		t.Fatal(err)
	}
	Vendored("tool", r, dir, "x refresh")

	want := "  [WARN] vendor - tool — not present — `x refresh` installs it\n" +
		"  [WARN] vendor - tool — not present — `x refresh` installs it\n" +
		"  [PASS] vendor - tool — present [1.2.3] — " + dir + "\n"
	if buf.String() != want {
		t.Errorf("printed\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestSizeReadsAsAPersonWould(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), make([]byte, 1536), 0644); err != nil {
		t.Fatal(err)
	}
	if got := Size(dir); got != "1.5 KB" {
		t.Errorf("Size = %q, want 1.5 KB", got)
	}
}
