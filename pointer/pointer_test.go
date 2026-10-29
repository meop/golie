package pointer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPointWritesThenMovesAPointerAndRemoveLeavesItsTarget(t *testing.T) {
	root := t.TempDir()
	one, two := filepath.Join(root, "1.0"), filepath.Join(root, "2.0")
	for _, d := range []string{one, two} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	vfile := filepath.Join(root, "version")

	if err := Point(vfile, one); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(vfile); !ok || got != one {
		t.Fatalf("Target = %q, %v", got, ok)
	}
	if err := Point(vfile, two); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(vfile); !ok || got != two {
		t.Fatalf("after a move, Target = %q, %v", got, ok)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 3 {
		t.Errorf("moving the pointer left %v", entries)
	}
	if err := Remove(vfile); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(two); err != nil {
		t.Error("Remove took the target")
	}
	if err := Remove(vfile); err != nil {
		t.Errorf("removing no pointer: %v", err)
	}
}

func TestPointNeverReplacesADirectoryOrTakesARelativeTarget(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "version")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Point(dir, root); err == nil {
		t.Error("a directory was replaced by a pointer")
	}
	if err := Remove(dir); err == nil {
		t.Error("Remove took a directory")
	}
	if err := Point(filepath.Join(root, "rel"), "1.0"); err == nil {
		t.Error("a relative target was pointed at")
	}
}

func TestAPointerNamesItsDirectoryRelativeToItsOwn(t *testing.T) {
	root := t.TempDir()
	vfile, dir := filepath.Join(root, "version"), filepath.Join(root, "0.74.5")
	if err := Point(vfile, dir); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(vfile); err != nil || string(b) != "0.74.5\n" {
		t.Errorf("the pointer holds %q, %v", b, err)
	}
	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(filepath.Join(moved, "version")); !ok || got != filepath.Join(moved, "0.74.5") {
		t.Errorf("after moving the state dir, Target = %q, %v", got, ok)
	}
}

func TestAnEmptyFileIsNotAPointer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, ok := Target(path); ok {
		t.Error("an empty file read as a pointer")
	}
	if err := Remove(path); err == nil {
		t.Error("Remove took a file that is not a pointer")
	}
}
