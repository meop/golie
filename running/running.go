// Package running says which programs are running from a directory, so a tool
// that removes an old version leaves the one a long-running program was started
// from. A server whose binary is gone loses more than its future: macOS's
// firewall drops connections to a program it can no longer read, and Windows
// will not delete a running program's file, so a removal stops halfway.
package running

import (
	"path/filepath"
	"strings"
)

// Programs are the executables of the processes this user can see, each by the
// path it was started from. A process the OS will not describe is left out; on
// an OS this package does not know, there are none.
func Programs() []string { return programs() }

// Under is the first of programs that is inside dir, or "". A path through a
// link is compared as the path it resolves to as well, on both sides.
func Under(programs []string, dir string) string {
	dir = filepath.Clean(dir)
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		resolvedDir = dir
	}
	for _, p := range programs {
		if within(filepath.Clean(p), dir) || within(filepath.Clean(p), resolvedDir) {
			return p
		}
	}
	for _, p := range programs {
		if resolved, err := filepath.EvalSymlinks(p); err == nil && within(resolved, resolvedDir) {
			return p
		}
	}
	return ""
}

func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
