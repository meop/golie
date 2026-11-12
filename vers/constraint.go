package vers

import (
	"fmt"
	"strconv"
	"strings"
)

type PinLevel int

const (
	PinMajor PinLevel = 1 // "14"     — updates within same major
	PinMinor PinLevel = 2 // "14.1"   — updates within same major.minor
	PinExact PinLevel = 3 // "14.1.0" — static, no updates
)

// Constraint represents a parsed version pin from user input.
type Constraint struct {
	Major int
	Minor int      // -1 if not specified
	Level PinLevel // 1, 2, or 3
	Raw   string   // normalised string, v stripped (e.g. "14", "14.1", "14.1.0")
}

func (l PinLevel) String() string {
	switch l {
	case PinMajor:
		return "major"
	case PinMinor:
		return "minor"
	case PinExact:
		return "fixed"
	}
	return "latest"
}

// ParseConstraint parses a version string into a Constraint. Any leading
// junk is stripped via SplitJunk — not just a "v" prefix — so a full
// literal release tag (e.g. bun's "bun-v1.2.3.4", llama.cpp's "b1234") works
// as a pin input too, not just a bare version.
func ParseConstraint(s string) (Constraint, error) {
	_, ver, trailing, ok := SplitJunk(s)
	if !ok {
		return Constraint{}, fmt.Errorf("invalid version %q: expected numeric major version", s)
	}
	s = ver + trailing
	parts := strings.Split(s, ".")
	c := Constraint{Raw: s, Minor: -1}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return c, fmt.Errorf("invalid version %q: expected numeric major version", s)
	}
	c.Major = major
	c.Level = PinMajor

	if len(parts) >= 2 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return c, fmt.Errorf("invalid version %q: expected numeric minor version", s)
		}
		c.Minor = minor
		c.Level = PinMinor
	}

	if len(parts) >= 3 {
		c.Level = PinExact
	}

	return c, nil
}
