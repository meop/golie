package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRollbackReversesNewestFirst(t *testing.T) {
	var order []string
	j := &Journal{}
	for _, what := range []string{"first", "second", "third"} {
		j.Did(what, func() error { order = append(order, what); return nil })
	}
	if errs := j.Rollback(); len(errs) != 0 {
		t.Fatalf("rollback errors: %v", errs)
	}
	if strings.Join(order, ",") != "third,second,first" {
		t.Errorf("undo order = %v", order)
	}
	if errs := j.Rollback(); len(errs) != 0 || len(order) != 3 || j.Len() != 0 {
		t.Errorf("a rolled back journal is spent, got %v after %v", order, errs)
	}
}

// the step that resists undoing is usually the locked file that failed the
// op, and stopping there would strand every earlier step half applied
func TestRollbackContinuesPastAFailingUndo(t *testing.T) {
	undone := 0
	j := &Journal{}
	j.Did("earlier", func() error { undone++; return nil })
	j.Did("locked", func() error { return errors.New("file in use") })
	errs := j.Rollback()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "locked: file in use") {
		t.Fatalf("errors = %v", errs)
	}
	if undone != 1 {
		t.Errorf("the earlier step was not undone")
	}
}

func TestDidDropsNilUndo(t *testing.T) {
	j := &Journal{}
	j.Did("nothing to reverse", nil)
	if j.Len() != 0 {
		t.Errorf("a nil undo recorded a step")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

func TestPlacePutsAVersionAtItsDirAndUndoRemovesIt(t *testing.T) {
	key := t.TempDir()
	stage, dir := filepath.Join(key, ".0.71.new"), filepath.Join(key, "0.71")
	write(t, filepath.Join(stage, "fzf"), "0.71")

	placed, undo, err := Place(stage, dir)
	if err != nil || placed != dir || read(t, filepath.Join(dir, "fzf")) != "0.71" {
		t.Fatalf("placed %s, %v", placed, err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, key); len(got) != 0 {
		t.Errorf("the undo left %v", got)
	}
}

func TestPlaceOfAVersionAlreadyThereGoesBesideItAndUndoRemovesOnlyTheNewOne(t *testing.T) {
	key := t.TempDir()
	dir := filepath.Join(key, "0.71")
	write(t, filepath.Join(dir, "fzf"), "running")
	stage := filepath.Join(key, ".0.71.new")
	write(t, filepath.Join(stage, "fzf"), "reinstalled")

	placed, undo, err := Place(stage, dir)
	if err != nil || placed == dir || !strings.HasPrefix(placed, dir+"+") {
		t.Fatalf("placed %s, %v", placed, err)
	}
	if read(t, filepath.Join(dir, "fzf")) != "running" || read(t, filepath.Join(placed, "fzf")) != "reinstalled" {
		t.Error("the version already there was replaced")
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, key); len(got) != 1 || got[0] != "0.71" || read(t, filepath.Join(dir, "fzf")) != "running" {
		t.Errorf("the undo left %v", got)
	}
}

func TestPointMovesTheLinkAndUndoMovesItBackOrRemovesIt(t *testing.T) {
	key := t.TempDir()
	one, two := filepath.Join(key, "0.70"), filepath.Join(key, "0.71")
	write(t, filepath.Join(one, "fzf"), "0.70")
	write(t, filepath.Join(two, "fzf"), "0.71")
	latest := filepath.Join(key, "latest")

	first, err := Point(latest, one)
	if err != nil {
		t.Fatal(err)
	}
	undo, err := Point(latest, two)
	if err != nil || read(t, filepath.Join(latest, "fzf")) != "0.71" {
		t.Fatalf("after the move: %v", err)
	}
	if err := undo(); err != nil || read(t, filepath.Join(latest, "fzf")) != "0.70" {
		t.Fatalf("the undo did not move it back: %v", err)
	}
	if err := first(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(latest); !os.IsNotExist(err) || read(t, filepath.Join(one, "fzf")) != "0.70" {
		t.Error("undoing the first link left it, or took its target")
	}
}
