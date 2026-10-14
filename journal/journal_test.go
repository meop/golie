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

func TestExtractUndo(t *testing.T) {
	// re-extracting over the installed version must not delete the install
	if ExtractUndo(t.TempDir(), "1.2.0", "1.2.0") != nil {
		t.Error("an undo for an in-place extraction")
	}
	dir := filepath.Join(t.TempDir(), "1.3.0")
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ExtractUndo(dir, "1.2.0", "1.3.0")(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("the new version dir survived its undo")
	}
}
