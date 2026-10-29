package link

import (
	"os"
	"path/filepath"
	"testing"
)

func dirWith(t *testing.T, parent, name, body string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "v"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func through(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(path, "v"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPointMakesThenMovesALinkAndRemoveLeavesItsTarget(t *testing.T) {
	root := t.TempDir()
	one, two := dirWith(t, root, "1.0", "one"), dirWith(t, root, "2.0", "two")
	current := filepath.Join(root, "current")

	if err := Point(current, one); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(current); !ok || got != one || through(t, current) != "one" {
		t.Fatalf("Target = %q, %v", got, ok)
	}
	if err := Point(current, two); err != nil {
		t.Fatal(err)
	}
	if got, ok := Target(current); !ok || got != two || through(t, current) != "two" {
		t.Fatalf("after a move, Target = %q, %v", got, ok)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 3 {
		t.Errorf("moving the link left %v", entries)
	}

	if err := Remove(current); err != nil {
		t.Fatal(err)
	}
	if _, ok := Target(current); ok || through(t, two) != "two" {
		t.Error("Remove took the target, or left the link")
	}
	if err := Remove(current); err != nil {
		t.Errorf("removing no link: %v", err)
	}
}

func TestPointNeverReplacesARealDirectoryOrTakesARelativeTarget(t *testing.T) {
	root := t.TempDir()
	real := dirWith(t, root, "current", "mine")
	if err := Point(real, dirWith(t, root, "1.0", "one")); err == nil || through(t, real) != "mine" {
		t.Error("a real directory was replaced by a link")
	}
	if err := Remove(real); err == nil {
		t.Error("Remove took a real directory")
	}
	if err := Point(filepath.Join(root, "rel"), "1.0"); err == nil {
		t.Error("a relative target was linked")
	}
	if _, ok := Target(real); ok {
		t.Error("a real directory read as a link")
	}
}
