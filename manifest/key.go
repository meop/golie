package manifest

import (
	"strings"

	"github.com/meop/golie/vers"
)

// The pin levels an Entry records.
const (
	PinLatest = "latest"
	PinMajor  = "major"
	PinMinor  = "minor"
	PinFixed  = "fixed"
)

// ParseKey splits "fzf@0.70" into ("fzf", "0.70", true). With no "@" it is
// (arg, "", false): the install that follows the newest release.
func ParseKey(arg string) (name, version string, pinned bool) {
	if before, after, found := strings.CutLast(arg, "@"); found {
		return before, after, true
	}
	return arg, "", false
}

// Key is what an install is recorded and listed under: the name alone when it
// follows the newest release, and the name with what it is pinned to
// otherwise.
func Key(name, version string) string {
	if version == "" {
		return name
	}
	return name + "@" + strings.TrimPrefix(version, "v")
}

// Pin is the level a pin holds an install to: "latest" for none, "major" for
// "14", "minor" for "14.1" and "fixed" for "14.1.0". A pin that is not
// numbers all the way ("1.28rc1") names one release, so it is fixed too.
func Pin(version string) string {
	if version == "" {
		return PinLatest
	}
	c, err := vers.ParseConstraint(version)
	if err != nil {
		return PinFixed
	}
	return c.Level.String()
}

// ShimName is what one of an install's binaries is called on PATH: its own
// name for the install that follows the newest release, and its name with the
// pin for any other ("fzf@14", "fzf@14.exe"), so pinned installs sit beside
// each other and none replaces another.
func ShimName(key, binName string) string {
	_, version, pinned := ParseKey(key)
	if !pinned {
		return binName
	}
	if base, ok := strings.CutSuffix(binName, ".exe"); ok {
		return base + "@" + version + ".exe"
	}
	return binName + "@" + version
}

// ParseBinPath splits a bin key into the directory inside the extract dir and
// the binary's file name.
func ParseBinPath(binKey string) (dir, name string) {
	dir, name, found := strings.CutLast(binKey, "/")
	if !found {
		return "", binKey
	}
	return dir, name
}
