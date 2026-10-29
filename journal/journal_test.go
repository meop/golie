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

func TestSwapPutsTheNewInstallAtTheSamePathAndUndoBringsTheOldBack(t *testing.T) {
	extracts := t.TempDir()
	live, stage := filepath.Join(extracts, "fzf"), filepath.Join(extracts, ".fzf.new")
	write(t, filepath.Join(live, "bin", "fzf"), "0.70")
	write(t, filepath.Join(stage, "bin", "fzf"), "0.71")

	aside, undo, err := Swap(stage, live)
	if err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(live, "bin", "fzf")); got != "0.71" {
		t.Errorf("live holds %s after the swap", got)
	}
	if got := read(t, filepath.Join(aside, "bin", "fzf")); got != "0.70" {
		t.Errorf("aside holds %s", got)
	}

	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(live, "bin", "fzf")); got != "0.70" {
		t.Errorf("live holds %s after the undo", got)
	}
	if got := entries(t, extracts); len(got) != 1 || got[0] != "fzf" {
		t.Errorf("the undo left %v", got)
	}
}

func TestSwapOfAFirstInstallHasNothingAsideAndUndoRemovesIt(t *testing.T) {
	extracts := t.TempDir()
	live, stage := filepath.Join(extracts, "fzf"), filepath.Join(extracts, ".fzf.new")
	write(t, filepath.Join(stage, "fzf"), "0.71")

	aside, undo, err := Swap(stage, live)
	if err != nil || aside != "" {
		t.Fatalf("aside %q, %v", aside, err)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, extracts); len(got) != 0 {
		t.Errorf("the undo left %v", got)
	}
}

func TestSwapThatCannotMoveTheStageInLeavesTheInstallAsItWas(t *testing.T) {
	extracts := t.TempDir()
	live := filepath.Join(extracts, "fzf")
	write(t, filepath.Join(live, "fzf"), "0.70")

	if _, _, err := Swap(filepath.Join(extracts, ".missing.new"), live); err == nil {
		t.Fatal("a swap with no stage succeeded")
	}
	if got := read(t, filepath.Join(live, "fzf")); got != "0.70" {
		t.Errorf("live holds %s", got)
	}
	if got := entries(t, extracts); len(got) != 1 {
		t.Errorf("the failed swap left %v", got)
	}
}
