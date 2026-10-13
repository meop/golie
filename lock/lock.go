// Package lock is the one-process-at-a-time lock a tool takes over its state
// directory before it changes anything in it.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// Check attempts a single non-blocking acquisition of the lock at path to
// detect whether another process of tool is running. It returns nil when the
// lock is free.
func Check(path, tool string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	fl := flock.New(path)
	ok, err := fl.TryLock()
	if err != nil {
		return fmt.Errorf("checking lock: %w", err)
	}
	if !ok {
		return fmt.Errorf("lock held — another %s process may be running", tool)
	}
	_ = fl.Unlock()
	return nil
}

// Acquire takes the lock at path, trying for a couple of seconds before
// giving up, and returns the function that releases it.
func Acquire(path, tool string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	fl := flock.New(path)
	for i := range 3 {
		ok, err := fl.TryLock()
		if err != nil {
			return nil, fmt.Errorf("acquiring lock: %w", err)
		}
		if ok {
			return func() { _ = fl.Unlock() }, nil
		}
		if i < 2 {
			time.Sleep(time.Second)
		}
	}
	return nil, fmt.Errorf("another %s process is running (lock held on %s)", tool, path)
}
