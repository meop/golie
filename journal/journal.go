// Package journal is how an install is made all or nothing.
//
// The manifest is the commit point: an entry is written only once a package's
// whole op succeeded, and anything short of that rolls the journal back, so
// the state directory is left exactly as the op found it and the untouched
// manifest still describes it. The alternative — writing what happened to
// survive — is what corrupts state: a sync whose shim create failed on a
// locked binary used to still record the new version and drop the shim it
// could not re-point, leaving a manifest that claimed a version that was not
// on disk and had forgotten a shim that was.
//
// This is also why the install a Swap moves aside is removed only after the
// manifest write: every undo here restores from it.
package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/meop/golie/shim"
)

// Journal records the filesystem mutations one package's op has already
// applied, each paired with the undo that reverses it.
type Journal struct {
	steps []step
}

type step struct {
	what string
	undo func() error
}

// Did records a mutation. The undo is recorded before the mutation is made:
// a step that fails can already have destroyed what it was replacing. A nil
// undo is dropped: some mutations (re-extracting over the dir already
// installed) have nothing to reverse, and the call sites read better saying
// so than branching.
func (j *Journal) Did(what string, undo func() error) {
	if undo == nil {
		return
	}
	j.steps = append(j.steps, step{what: what, undo: undo})
}

// Len is how many steps are recorded.
func (j *Journal) Len() int { return len(j.steps) }

// Rollback reverses every recorded step, newest first, and returns one error
// per step it could not undo. It never stops early: the step that resists
// undoing is typically the very thing that failed the op (a locked file), and
// letting it strand the steps recorded before it would leave exactly the
// half-applied state rolling back exists to prevent.
func (j *Journal) Rollback() []error {
	var errs []error
	for _, s := range slices.Backward(j.steps) {
		if err := s.undo(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.what, err))
		}
	}
	j.steps = nil
	return errs
}

// Swap moves live aside and stage into its place, and returns the undo. aside
// is empty for a first install, and otherwise the caller's to remove after
// the manifest write. Windows moves a directory with a binary of it running.
func Swap(stage, live string) (aside string, undo func() error, err error) {
	_, statErr := os.Lstat(live)
	had := statErr == nil
	if had {
		aside = beside(live, "old")
		if err := os.Rename(live, aside); err != nil {
			return "", nil, err
		}
	}
	if err := os.Rename(stage, live); err != nil {
		if had {
			_ = os.Rename(aside, live)
		}
		return "", nil, err
	}
	return aside, func() error {
		failed := beside(live, "failed")
		if err := os.Rename(live, failed); err != nil {
			return err
		}
		_ = os.RemoveAll(failed)
		if had {
			return os.Rename(aside, live)
		}
		return nil
	}, nil
}

// beside is unique, so one a running binary kept from removal never blocks the next.
func beside(path, what string) string {
	return filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.%d.%s", filepath.Base(path), time.Now().UnixNano(), what))
}

// ShimUndo reverses a shim about to be stamped or deleted. It reads the shim's
// target when called, so it is called before the mutation.
func ShimUndo(s shim.Stamper, file string) func() error {
	path := s.Path(file)
	if old, ok := shim.StampedTarget(path); ok && s.IsShim(path) {
		return func() error { return s.Create(file, old, true) }
	}
	return func() error { return s.Remove(file) }
}
