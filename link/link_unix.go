//go:build !windows

package link

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func create(path, dir string) error { return os.Symlink(dir, path) }

func replace(path, dir string) error {
	tmp := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.%d.link", filepath.Base(path), time.Now().UnixNano()))
	if err := os.Symlink(dir, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
