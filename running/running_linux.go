package running

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func programs() []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		exe, err := os.Readlink(filepath.Join("/proc", e.Name(), "exe"))
		if err != nil {
			continue
		}
		out = append(out, strings.TrimSuffix(exe, " (deleted)"))
	}
	return out
}
