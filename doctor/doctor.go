// Package doctor prints a tool's doctor command, the same in every tool built
// on golie: the tool's name over a rule, then one line a check,
// `[PASS] label — detail`, and a vendored tool labelled and versioned as
// its refresh says it.
package doctor

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/meop/golie/ui"
	"github.com/meop/golie/vendored"
)

// Start prints the title a doctor command opens with.
func Start(tool string) {
	ui.Out("%s doctor", tool)
	ui.Out("%s", strings.Repeat("─", 50))
}

// Pass, Warn and Fail print one check's line.
func Pass(label, detail string) { line("pass", "PASS", label, detail) }
func Warn(label, detail string) { line("warn", "WARN", label, detail) }
func Fail(label, detail string) { line("fail", "FAIL", label, detail) }

func line(role, status, label, detail string) {
	if detail == "" {
		ui.Out("  [%s] %s", ui.Color(role, status), label)
		return
	}
	ui.Out("  [%s] %s — %s", ui.Color(role, status), label, detail)
}

// Present is the detail of a check that found what it looked for: its
// version, in brackets as refresh says it, and where it is.
func Present(version, path string) string {
	return fmt.Sprintf("present [%s] — %s", version, path)
}

// Vendored checks that dir holds a whole release of the tool name, labelled
// as refresh labels it, and says which, or that refresh, the command named,
// puts one there.
func Vendored(name string, r vendored.Release, dir, refresh string) {
	if v := r.Installed(dir); v != "" {
		Pass(vendored.Label(name), Present(v, dir))
		return
	}
	Warn(vendored.Label(name), fmt.Sprintf("not present — `%s` installs it", refresh))
}

// Size is how much the files under path take, as a person reads it.
func Size(path string) string {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	const unit = 1024
	if total < unit {
		return fmt.Sprintf("%d B", total)
	}
	div, exp := int64(unit), 0
	for n := total / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(total)/float64(div), "KMGTPE"[exp])
}
