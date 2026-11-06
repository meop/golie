package running

import (
	"os"
	"path/filepath"
	"testing"
)

// the test binary is a running program like any other, on every OS this ships for
func TestProgramsHasThisOne(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if Under(Programs(), filepath.Dir(self)) == "" {
		t.Fatalf("no running program under %s, though %s runs from it", filepath.Dir(self), self)
	}
}

// a version dir whose name another starts with is not the other's
func TestUnder(t *testing.T) {
	root := t.TempDir()
	program := filepath.Join(root, "delish", "0.6.0", "delishd")
	for _, tc := range []struct {
		dir  string
		want string
	}{
		{filepath.Join(root, "delish", "0.6.0"), program},
		{filepath.Join(root, "delish"), program},
		{filepath.Join(root, "delish", "0.6"), ""},
		{filepath.Join(root, "delish", "0.6.0", "delishd"), ""},
		{filepath.Join(root, "other"), ""},
	} {
		if got := Under([]string{program}, tc.dir); got != tc.want {
			t.Errorf("Under(%s) = %q, want %q", tc.dir, got, tc.want)
		}
	}
}

// a program is named by the path it was started from, which can reach its dir through a link the dir asked about
// does not, or the other way around: macOS's temp dir is one, and so is Windows' short name for a long one
func TestUnderThroughALink(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	if err := os.MkdirAll(filepath.Join(real, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "bin", "tool"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ program, dir string }{
		{filepath.Join(real, "bin", "tool"), link},
		{filepath.Join(link, "bin", "tool"), real},
	} {
		if got := Under([]string{tc.program}, tc.dir); got != tc.program {
			t.Errorf("Under(%s) with %s running = %q", tc.dir, tc.program, got)
		}
	}
}
