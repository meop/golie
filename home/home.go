// Package home resolves the user's home directory for a tool that keeps its
// state under it, with a guard that keeps that tool's tests out of the real
// one.
package home

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns the user's real home in production and an explicitly isolated
// home when the environment variable testEnv names one. Test binaries refuse
// to fall back to the real home: a missed test setup must fail before it can
// modify what a user has installed.
func Dir(testEnv string) (string, error) {
	if dir := os.Getenv(testEnv); dir != "" {
		return filepath.Clean(dir), nil
	}
	if runningUnderGoTest() {
		return "", fmt.Errorf("refusing to use the real user home during tests: set %s", testEnv)
	}
	return os.UserHomeDir()
}

// UsingTest reports whether Dir is redirected by testEnv. Callers with
// OS-specific paths, such as LOCALAPPDATA, use this to keep every derived
// path under the same isolated root.
func UsingTest(testEnv string) bool {
	return os.Getenv(testEnv) != ""
}

func runningUnderGoTest() bool {
	if flag.Lookup("test.v") != nil {
		return true
	}
	name := strings.ToLower(filepath.Base(os.Args[0]))
	return strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".test.exe")
}
