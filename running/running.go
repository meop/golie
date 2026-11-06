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

// Under is the first of programs that is inside dir, or "".
func Under(programs []string, dir string) string {
	dirs := []string{filepath.Clean(dir)}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved != dirs[0] {
		dirs = append(dirs, resolved)
	}
	for _, p := range programs {
		for _, d := range dirs {
			if within(filepath.Clean(p), d) {
				return p
			}
		}
	}
	return ""
}

func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
