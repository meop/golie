// Package manifest is the record a tool keeps of what it installed: one file,
// one entry per install, written only once an install is whole. ghpm and gotm
// keep the same file, so everything that reads it — list, sync, remove, tidy —
// reads one contract.
package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

// Entry records one install. Its release assets are overlaid (in Assets
// order, later wins on a path collision) into a single extract dir, so bins
// and fonts are flat maps over that combined tree rather than per asset.
// Assets is kept so the same set can be downloaded and overlaid again.
//
// Pin is how far the install may move — "latest", "major", "minor" or
// "fixed" — and Version is the release it is on now. What it is pinned to is
// in its key ("fzf@14"), not here.
//
// BinDeclined and FontDeclined are the discovered artifacts the user did not
// select. With the selected ones they are everything discovered at install
// time, which is what tells "the user wanted a subset" from "the release
// gained a binary" the next time the package moves.
type Entry struct {
	Pin          string            `json:"pin"`
	Version      string            `json:"version"`
	Assets       []string          `json:"assets,omitempty"`
	Bin          map[string]string `json:"bin,omitempty"`
	Font         map[string]string `json:"font,omitempty"`
	BinDeclined  []string          `json:"bin_declined,omitempty"`
	FontDeclined []string          `json:"font_declined,omitempty"`
}

// AllBins is the entry's shim name → bin key map. A bin key is the binary's
// path inside the extract dir, slash separated.
func (e Entry) AllBins() map[string]string { return e.Bin }

// AllFonts is the entry's font name → font path map.
func (e Entry) AllFonts() map[string]string { return e.Font }

// DiscoveredBins is every bin key discovered at install time: the selected
// ones and the declined ones. An entry written before declines were recorded
// has only the selected.
func (e Entry) DiscoveredBins() []string {
	keys := make([]string, 0, len(e.Bin)+len(e.BinDeclined))
	for _, binKey := range e.Bin {
		keys = append(keys, binKey)
	}
	return append(keys, e.BinDeclined...)
}

// DiscoveredFonts is DiscoveredBins for fonts.
func (e Entry) DiscoveredFonts() []string {
	paths := make([]string, 0, len(e.Font)+len(e.FontDeclined))
	for _, fontPath := range e.Font {
		paths = append(paths, fontPath)
	}
	return append(paths, e.FontDeclined...)
}

// Manifest is everything a tool records. Repos maps a package name to where
// its releases come from; Extracts maps a key — the name, or the name and
// what it is pinned to ("fzf@14") — to the install.
type Manifest struct {
	Repos    map[string]string `json:"repo"`
	Extracts map[string]Entry  `json:"extract"`
}

// New is a manifest with nothing installed.
func New() *Manifest {
	return &Manifest{Repos: map[string]string{}, Extracts: map[string]Entry{}}
}

// Keys lists the installed keys, sorted.
func (m *Manifest) Keys() []string {
	keys := make([]string, 0, len(m.Extracts))
	for k := range m.Extracts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// RemoveExtract forgets an install, and its package's source once no other
// install of that package is left.
func (m *Manifest) RemoveExtract(key string) {
	delete(m.Extracts, key)
	name, _, _ := ParseKey(key)
	for k := range m.Extracts {
		if n, _, _ := ParseKey(k); n == name {
			return
		}
	}
	delete(m.Repos, name)
}

// FindBySource returns the name already registered with source.
func (m *Manifest) FindBySource(source string) (string, bool) {
	for name, src := range m.Repos {
		if src == source {
			return name, true
		}
	}
	return "", false
}

// Load reads the manifest at path, or returns an empty one when there is
// none yet.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Repos == nil {
		m.Repos = map[string]string{}
	}
	if m.Extracts == nil {
		m.Extracts = map[string]Entry{}
	}
	return &m, nil
}

// Save writes the manifest through a temp file and a rename, so a crash
// mid-write leaves the old one in place rather than a truncated one.
func Save(m *Manifest, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
