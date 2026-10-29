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
	latest := filepath.Join(root, "latest")

	if err := Point(latest, one); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(latest); !ok || got != one {
		t.Fatalf("Target = %q, %v", got, ok)
	}
	if err := Point(latest, two); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(latest); !ok || got != two {
		t.Fatalf("after a move, Target = %q, %v", got, ok)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 3 {
		t.Errorf("moving the pointer left %v", entries)
	}
	if err := Remove(latest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(two); err != nil {
		t.Error("Remove took the target")
	}
	if err := Remove(latest); err != nil {
		t.Errorf("removing no pointer: %v", err)
	}
}

func TestPointNeverReplacesADirectoryOrTakesARelativeTarget(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "latest")
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
	latest, dir := filepath.Join(root, "latest"), filepath.Join(root, "0.74.5")
	if err := Point(latest, dir); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(latest); err != nil || string(b) != "0.74.5\n" {
		t.Errorf("the pointer holds %q, %v", b, err)
	}
	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(filepath.Join(moved, "latest")); !ok || got != filepath.Join(moved, "0.74.5") {
		t.Errorf("after moving the state dir, Target = %q, %v", got, ok)
	}
}

func TestAnEmptyFileIsNotAPointer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latest")
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
