// Package pointer writes and reads pointer files: a regular file naming a
// directory relative to its own, which a wedge shim stamped through it reads
// as that directory.
package pointer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Point makes path a pointer to dir, replacing a pointer already there.
func Point(path, dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("pointer target %s is not absolute", dir)
	}
	rel, err := filepath.Rel(filepath.Dir(path), dir)
	if err != nil {
		return err
	}
	if fi, err := os.Lstat(path); err == nil && !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a pointer", path)
	}
	tmp := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.%d.tmp", filepath.Base(path), time.Now().UnixNano()))
	if err := os.WriteFile(tmp, []byte(filepath.ToSlash(rel)+"\n"), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Target is the directory path points at; false when path is not a pointer.
func Target(path string) (string, bool) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > 4096 {
		return "", false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	named := strings.TrimSpace(string(b))
	if named == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(path), filepath.FromSlash(named)), true
}

// Remove removes the pointer at path, never what it points at.
func Remove(path string) error {
	if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if _, ok := Target(path); !ok {
		return fmt.Errorf("%s is not a pointer", path)
	}
	return os.Remove(path)
}
