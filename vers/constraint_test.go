package vers

import (
	"testing"
)

func TestParseConstraint(t *testing.T) {
	cases := []struct {
		input string
		major int
		minor int
		level PinLevel
		raw   string
		err   bool
	}{
		{"14", 14, -1, PinMajor, "14", false},
		{"v14", 14, -1, PinMajor, "14", false},
		{"14.1", 14, 1, PinMinor, "14.1", false},
		{"14.1.0", 14, 1, PinExact, "14.1.0", false},
		{"14.1.2", 14, 1, PinExact, "14.1.2", false},
		{"0.56.0", 0, 56, PinExact, "0.56.0", false},
		{"v0.56.0", 0, 56, PinExact, "0.56.0", false},
		// A full literal release tag works as a pin input too, not just a
		// bare version — any leading junk is stripped, not only "v".
		{"bun-v1.2.3.4", 1, 2, PinExact, "1.2.3.4", false},
		{"b1234", 1234, -1, PinMajor, "1234", false},
		{"notaversion", 0, 0, 0, "", true},
	}
	for _, c := range cases {
		got, err := ParseConstraint(c.input)
		if c.err {
			if err == nil {
				t.Errorf("ParseConstraint(%q) expected error", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseConstraint(%q) unexpected error: %v", c.input, err)
			continue
		}
		if got.Major != c.major || got.Minor != c.minor || got.Level != c.level || got.Raw != c.raw {
			t.Errorf("ParseConstraint(%q) = {%d,%d,%d,%q}, want {%d,%d,%d,%q}",
				c.input, got.Major, got.Minor, int(got.Level), got.Raw,
				c.major, c.minor, int(c.level), c.raw)
		}
	}
}
