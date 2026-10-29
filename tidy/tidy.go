// Package tidy finds what a state directory holds that its manifest does not
// account for, and what the manifest claims that the disk no longer bears
// out, and removes either. It prints nothing and asks nothing: a tool shows
// what a Find returned, gets its confirmation, and calls the removal.
//
// Removals commit the way an uninstall does. Nothing here can be rolled back,
// so the order is the safety: a file that will not delete keeps its manifest
// entry rather than being forgotten with the file still there.
package tidy

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/meop/golie/layout"
	"github.com/meop/golie/manifest"
	"github.com/meop/golie/pointer"
)

// Dirs is the part of a state directory tidy looks at.
type Dirs struct {
	Bin       string
	Extracts  string
	Downloads string
}

// DirsOf is the Dirs of a layout.
func DirsOf(l layout.Layout) (Dirs, error) {
	bin, err := l.BinDir()
	if err != nil {
		return Dirs{}, err
	}
	return Dirs{Bin: bin, Extracts: l.ExtractsDir(), Downloads: l.DownloadsDir()}, nil
}

// Broken is something a manifest entry claims that is not on disk: its
// extract dir, or one of its shims.
type Broken struct {
	Key   string
	Issue string

	shimPaths   []string
	extractPath string
	trimShims   []string
}

// Stuck is a file that would not delete.
type Stuck struct {
	Path string
	Err  error
}

// FindBroken lists the installs the disk no longer bears out, by key. An
// install with its extract dir gone is one item; an install that is only
// missing shims is one item per shim, since each is trimmed on its own.
func FindBroken(m *manifest.Manifest, d Dirs) []Broken {
	var items []Broken
	for _, key := range m.Keys() {
		e := m.Extracts[key]
		extractPath := filepath.Join(d.Extracts, key)
		_, err := os.Lstat(extractPath)
		extractMissing := os.IsNotExist(err)
		if target, ok := pointer.Target(filepath.Join(extractPath, layout.Latest)); ok {
			_, err := os.Stat(target)
			extractMissing = os.IsNotExist(err)
		}

		if len(e.Bin) == 0 {
			if len(e.Font) > 0 && extractMissing {
				items = append(items, Broken{Key: key, Issue: "missing extract", extractPath: extractPath})
			}
			continue
		}
		var shimPaths, missing []string
		for shimName := range e.Bin {
			sp := filepath.Join(d.Bin, shimName)
			shimPaths = append(shimPaths, sp)
			if _, err := os.Lstat(sp); os.IsNotExist(err) {
				missing = append(missing, shimName)
			}
		}
		slices.Sort(shimPaths)
		slices.Sort(missing)
		if extractMissing {
			items = append(items, Broken{Key: key, Issue: "missing extract", shimPaths: shimPaths, extractPath: extractPath})
			continue
		}
		for _, shimName := range missing {
			items = append(items, Broken{Key: key, Issue: fmt.Sprintf("missing bin (%s)", shimName), trimShims: []string{shimName}})
		}
	}
	return items
}

// RemoveBroken makes the manifest true again for one item, and reports
// whether it changed m. An install with no extract dir loses its shims, its
// cached download and its entry; one missing a shim forgets that shim, and
// goes entirely once it has no shim and no font left. A shim that will not
// delete is returned and nothing else is touched, so the entry still
// describes it for the next run.
func RemoveBroken(m *manifest.Manifest, b Broken, d Dirs) (changed bool, stuck []Stuck) {
	for _, sp := range b.shimPaths {
		if err := os.Remove(sp); err != nil && !os.IsNotExist(err) {
			stuck = append(stuck, Stuck{Path: sp, Err: err})
		}
	}
	if len(stuck) > 0 {
		return false, stuck
	}
	if b.extractPath != "" {
		_ = PruneExtract(b.extractPath, d.Extracts)
	}
	e := m.Extracts[b.Key]
	if len(b.trimShims) > 0 {
		for _, shimName := range b.trimShims {
			delete(e.Bin, shimName)
		}
		m.Extracts[b.Key] = e
		if len(e.Bin) == 0 && len(e.Font) == 0 {
			m.RemoveExtract(b.Key)
			_ = PruneExtract(filepath.Join(d.Extracts, b.Key), d.Extracts)
		}
		return true, nil
	}
	name, _, _ := manifest.ParseKey(b.Key)
	if src, ok := m.Repos[name]; ok {
		PruneDownloadVersion(d.Downloads, src, e.Version)
	}
	m.RemoveExtract(b.Key)
	return true, nil
}

// FindOrphanShims lists the files in binDir that isShim says the tool
// stamped and no manifest entry names. keep names files that are the tool's
// without being an install's: itself. Anything isShim does not vouch for is
// somebody else's and is never listed.
func FindOrphanShims(m *manifest.Manifest, binDir string, keep []string, isShim func(path string) bool) []string {
	expected := map[string]bool{}
	for _, name := range keep {
		expected[name] = true
	}
	for _, e := range m.Extracts {
		for shimName := range e.Bin {
			expected[shimName] = true
		}
	}
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		full := filepath.Join(binDir, e.Name())
		if !expected[e.Name()] && isShim(full) {
			paths = append(paths, full)
		}
	}
	return paths
}

// OrphanExtract is an unpacked install no manifest entry is on, or what an op
// left staged or aside (Leftover).
type OrphanExtract struct {
	Key      string
	Leftover bool
	Path     string
}

var leftover = regexp.MustCompile(`^\.(.+?)(\.new|\.\d+\.(old|failed))$`)

// FindOrphanExtracts lists what no install is on: extract dirs the manifest
// does not name, leftovers, and inside an install all but latest and its target.
func FindOrphanExtracts(m *manifest.Manifest, extractsDir string) []OrphanExtract {
	entries, err := os.ReadDir(extractsDir)
	if err != nil {
		return nil
	}
	var orphans []OrphanExtract
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(extractsDir, e.Name())
		if match := leftover.FindStringSubmatch(e.Name()); match != nil {
			orphans = append(orphans, OrphanExtract{Key: match[1], Leftover: true, Path: path})
			continue
		}
		if _, installed := m.Extracts[e.Name()]; !installed {
			orphans = append(orphans, OrphanExtract{Key: e.Name(), Path: path})
			continue
		}
		target, ok := pointer.Target(filepath.Join(path, layout.Latest))
		if !ok {
			continue
		}
		inner, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		for _, v := range inner {
			if v.Name() != layout.Latest && v.Name() != filepath.Base(target) {
				orphans = append(orphans, OrphanExtract{Key: e.Name(), Leftover: true, Path: filepath.Join(path, v.Name())})
			}
		}
	}
	return orphans
}

// PruneInstall removes from an install's dir all but its latest pointer and the
// version it names, and returns what would not go.
func PruneInstall(keyDir string) (stuck []Stuck) {
	target, ok := pointer.Target(filepath.Join(keyDir, layout.Latest))
	if !ok {
		return nil
	}
	entries, err := os.ReadDir(keyDir)
	if err != nil {
		return []Stuck{{Path: keyDir, Err: err}}
	}
	for _, e := range entries {
		if e.Name() == layout.Latest || e.Name() == filepath.Base(target) {
			continue
		}
		path := filepath.Join(keyDir, e.Name())
		if err := os.RemoveAll(path); err != nil {
			stuck = append(stuck, Stuck{Path: path, Err: err})
		}
	}
	return stuck
}

// PruneExtract removes path, an extract dir or a leftover, never extractsDir.
func PruneExtract(path, extractsDir string) error {
	if filepath.Clean(path) == filepath.Clean(extractsDir) {
		return fmt.Errorf("refusing to remove the extracts dir %s", extractsDir)
	}
	return os.RemoveAll(path)
}

// OrphanDownload is a cached release asset of a version no install is on.
type OrphanDownload struct {
	Source  string
	Version string
	File    string
	Path    string
}

// Package is the last element of the download's source: the repo of
// "github.com/owner/repo".
func (o OrphanDownload) Package() string { return path.Base(o.Source) }

// FindOrphanDownloads lists the cached assets under downloadsDir whose
// source and version no manifest entry is on. depth is how many path elements
// a source has there (3 for "github.com/owner/repo"), which is what says
// where a source ends and its version dir begins.
func FindOrphanDownloads(m *manifest.Manifest, downloadsDir string, depth int) []OrphanDownload {
	installed := map[string]bool{}
	for key, e := range m.Extracts {
		name, _, _ := manifest.ParseKey(key)
		if src, ok := m.Repos[name]; ok {
			installed[src+"/"+e.Version] = true
		}
	}
	var orphans []OrphanDownload
	_ = filepath.WalkDir(downloadsDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(downloadsDir, p)
		if err != nil {
			return nil
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < depth+2 {
			return nil
		}
		source, version := strings.Join(parts[:depth], "/"), parts[depth]
		if !installed[source+"/"+version] {
			orphans = append(orphans, OrphanDownload{Source: source, Version: version, File: parts[len(parts)-1], Path: p})
		}
		return nil
	})
	return orphans
}

// PruneDownloadVersion removes the cached assets of one version of one
// source, and then every directory above them that is left empty, stopping at
// downloadsDir.
func PruneDownloadVersion(downloadsDir, source, version string) {
	parent := filepath.Join(downloadsDir, layout.SourceToRelPath(source))
	_ = os.RemoveAll(filepath.Join(parent, version))
	for parent != downloadsDir {
		es, err := os.ReadDir(parent)
		if err != nil || len(es) > 0 {
			break
		}
		_ = os.Remove(parent)
		parent = filepath.Dir(parent)
	}
}

// PruneEmptyDirs removes every empty directory under root, deepest first,
// and leaves root itself.
func PruneEmptyDirs(root string) {
	var dirs []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == root || !d.IsDir() {
			return nil
		}
		dirs = append(dirs, p)
		return nil
	})
	for _, dir := range slices.Backward(dirs) {
		if entries, _ := os.ReadDir(dir); len(entries) == 0 {
			_ = os.Remove(dir)
		}
	}
}
