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
// This is also why the version a Point moves the latest pointer off is removed
// only after the manifest write: every undo here restores from it.
package journal

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/meop/golie/pointer"
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

// Place moves stage to dir, or beside a dir already there as dir+<n>, and
// returns where it went and the undo.
func Place(stage, dir string) (placed string, undo func() error, err error) {
	placed = dir
	if _, err := os.Lstat(dir); err == nil {
		placed = fmt.Sprintf("%s+%d", dir, time.Now().UnixNano())
	}
	if err := os.Rename(stage, placed); err != nil {
		return "", nil, err
	}
	return placed, func() error { return os.RemoveAll(placed) }, nil
}

// Point moves the pointer at path to dir, and returns the undo.
func Point(path, dir string) (undo func() error, err error) {
	old, had := pointer.Target(path)
	if err := pointer.Point(path, dir); err != nil {
		return nil, err
	}
	return func() error {
		if had {
			return pointer.Point(path, old)
		}
		return pointer.Remove(path)
	}, nil
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
