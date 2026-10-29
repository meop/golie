// Package link points a fixed path at a directory: an absolute symlink on
// unix, a junction on Windows.
package link

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Point makes path a link to dir, replacing only a link.
func Point(path, dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("link target %s is not absolute", dir)
	}
	fi, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return create(path, dir)
	case err != nil:
		return err
	case !isLink(fi):
		return fmt.Errorf("%s is not a link", path)
	}
	return replace(path, dir)
}

// Target is the directory path links to; false when path is not a link.
func Target(path string) (string, bool) {
	fi, err := os.Lstat(path)
	if err != nil || !isLink(fi) {
		return "", false
	}
	dir, err := os.Readlink(path)
	if err != nil {
		return "", false
	}
	return dir, true
}

// Remove removes the link at path, never what it points at.
func Remove(path string) error {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isLink(fi) {
		return fmt.Errorf("%s is not a link", path)
	}
	return os.Remove(path)
}

// a junction reads as irregular rather than as a symlink
func isLink(fi fs.FileInfo) bool {
	return fi.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0
}
