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
// This is also why the old extract dir is removed only after the manifest
// write: every undo here restores from it.
package journal

import (
	"fmt"
	"os"
	"slices"

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

// ExtractUndo reverses an extraction into newDir. An op that lands on the
// version already installed re-extracts in place, so there is nothing to undo
// — and removing that dir would delete the live install.
func ExtractUndo(newDir, oldVersion, newVersion string) func() error {
	if oldVersion == newVersion {
		return nil
	}
	return func() error { return os.RemoveAll(newDir) }
}

// ShimUndo reverses a shim about to be stamped or deleted: re-stamping it at
// the binary it ran before the op, or removing it when the op is what created
// it. An empty oldSource means there was no such shim.
func ShimUndo(s shim.Stamper, file, oldSource string) func() error {
	if oldSource == "" {
		return func() error { return s.Remove(file) }
	}
	return func() error { return s.Create(file, oldSource, true) }
}
