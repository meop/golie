package tidy

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/meop/golie/layout"
	"github.com/meop/golie/manifest"
	"github.com/meop/golie/pointer"
)

// state is a state directory with fzf (two shims) and fzf@14 installed.
func state(t *testing.T) (*manifest.Manifest, Dirs) {
	t.Helper()
	l := layout.Layout{Root: filepath.Join(t.TempDir(), ".tool")}
	d, err := DirsOf(l)
	if err != nil {
		t.Fatal(err)
	}
	m := manifest.New()
	m.Repos["fzf"] = "github.com/junegunn/fzf"
	install(t, m, d, "fzf", "0.70.0", "fzf", "fzf-tmux")
	install(t, m, d, "fzf@14", "14.2.0", "fzf@14")
	return m, d
}

func install(t *testing.T, m *manifest.Manifest, d Dirs, key, version string, shims ...string) {
	t.Helper()
	bins := map[string]string{}
	for _, s := range shims {
		touch(t, filepath.Join(d.Bin, s))
		bins[s] = "bin/" + s
	}
	touch(t, filepath.Join(d.Extracts, key, "bin", "x"))
	name, _, _ := manifest.ParseKey(key)
	touch(t, filepath.Join(d.Downloads, layout.SourceToRelPath(m.Repos[name]), version, "asset.tar.gz"))
	m.Extracts[key] = manifest.Entry{Version: version, Bin: bins}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestHealthyStateHasNothingToTidy(t *testing.T) {
	m, d := state(t)
	if got := FindBroken(m, d); len(got) != 0 {
		t.Errorf("FindBroken = %+v", got)
	}
	if got := FindOrphanShims(m, d.Bin, nil, func(string) bool { return true }); len(got) != 0 {
		t.Errorf("FindOrphanShims = %v", got)
	}
	if got := FindOrphanExtracts(m, d.Extracts); len(got) != 0 {
		t.Errorf("FindOrphanExtracts = %+v", got)
	}
	if got := FindOrphanDownloads(m, d.Downloads, 3); len(got) != 0 {
		t.Errorf("FindOrphanDownloads = %+v", got)
	}
}

// an install whose extract dir is gone loses its shims, its download and its
// entry, and its package's source only with the last install of it
func TestBrokenMissingExtract(t *testing.T) {
	m, d := state(t)
	if err := os.RemoveAll(filepath.Join(d.Extracts, "fzf@14")); err != nil {
		t.Fatal(err)
	}
	items := FindBroken(m, d)
	if len(items) != 1 || items[0].Key != "fzf@14" || items[0].Issue != "missing extract" {
		t.Fatalf("FindBroken = %+v", items)
	}
	changed, stuck := RemoveBroken(m, items[0], d)
	if !changed || len(stuck) != 0 {
		t.Fatalf("RemoveBroken = %v, %v", changed, stuck)
	}
	if _, ok := m.Extracts["fzf@14"]; ok || exists(filepath.Join(d.Bin, "fzf@14")) {
		t.Error("the broken install was not removed")
	}
	if exists(filepath.Join(d.Downloads, "github.com", "junegunn", "fzf", "14.2.0")) {
		t.Error("its cached download was kept")
	}
	if _, ok := m.Repos["fzf"]; !ok || !exists(filepath.Join(d.Bin, "fzf")) {
		t.Error("the other install of the package was touched")
	}
}

// an install missing one shim forgets that shim and stays; missing all of
// them it goes, extract dir and all
func TestBrokenMissingShims(t *testing.T) {
	m, d := state(t)
	if err := os.Remove(filepath.Join(d.Bin, "fzf-tmux")); err != nil {
		t.Fatal(err)
	}
	items := FindBroken(m, d)
	if len(items) != 1 || items[0].Issue != "missing bin (fzf-tmux)" {
		t.Fatalf("FindBroken = %+v", items)
	}
	if changed, _ := RemoveBroken(m, items[0], d); !changed {
		t.Fatal("nothing changed")
	}
	if e := m.Extracts["fzf"]; len(e.Bin) != 1 || e.Bin["fzf"] == "" {
		t.Fatalf("entry = %+v", e)
	}

	if err := os.Remove(filepath.Join(d.Bin, "fzf")); err != nil {
		t.Fatal(err)
	}
	for _, it := range FindBroken(m, d) {
		RemoveBroken(m, it, d)
	}
	if _, ok := m.Extracts["fzf"]; ok || exists(filepath.Join(d.Extracts, "fzf")) {
		t.Error("an install with no shim left was kept")
	}
}

// a font-only install has no shims to miss, only its extract dir
func TestBrokenFontOnly(t *testing.T) {
	m, d := state(t)
	m.Extracts["font"] = manifest.Entry{Version: "1.0", Font: map[string]string{"F": "f.ttf"}}
	items := FindBroken(m, d)
	if len(items) != 1 || items[0].Key != "font" || items[0].Issue != "missing extract" {
		t.Fatalf("FindBroken = %+v", items)
	}
}

func TestOrphanShims(t *testing.T) {
	m, d := state(t)
	for _, name := range []string{"tool", "stray", "foreign"} {
		touch(t, filepath.Join(d.Bin, name))
	}
	mine := func(path string) bool { return filepath.Base(path) != "foreign" }
	got := FindOrphanShims(m, d.Bin, []string{"tool"}, mine)
	if len(got) != 1 || filepath.Base(got[0]) != "stray" {
		t.Errorf("FindOrphanShims = %v: the tool itself and a file it did not stamp are not orphans", got)
	}
}

func TestOrphanExtracts(t *testing.T) {
	m, d := state(t)
	touch(t, filepath.Join(d.Extracts, "gone", "x"))
	touch(t, filepath.Join(d.Extracts, ".fzf.new", "x"))
	touch(t, filepath.Join(d.Extracts, ".fzf@14.1793253698000000000.old", "x"))
	got := FindOrphanExtracts(m, d.Extracts)
	slices.SortFunc(got, func(a, b OrphanExtract) int { return strings.Compare(a.Path, b.Path) })
	want := []OrphanExtract{
		{Key: "fzf", Leftover: true, Path: filepath.Join(d.Extracts, ".fzf.new")},
		{Key: "fzf@14", Leftover: true, Path: filepath.Join(d.Extracts, ".fzf@14.1793253698000000000.old")},
		{Key: "gone", Path: filepath.Join(d.Extracts, "gone")},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("FindOrphanExtracts = %+v, want %+v", got, want)
	}
	for _, o := range got {
		if err := PruneExtract(o.Path, d.Extracts); err != nil {
			t.Fatal(err)
		}
	}
	if len(FindOrphanExtracts(m, d.Extracts)) != 0 || !exists(filepath.Join(d.Extracts, "fzf")) {
		t.Error("pruning took the wrong dirs")
	}
	if err := PruneExtract(d.Extracts, d.Extracts); err == nil || !exists(d.Extracts) {
		t.Error("the extracts dir itself was pruned")
	}
}

// where a source ends under download/ is the tool's to say: three elements
// for a github repo, two for go.dev/dl
func TestOrphanDownloads(t *testing.T) {
	m, d := state(t)
	touch(t, filepath.Join(d.Downloads, "github.com", "junegunn", "fzf", "0.69.0", "old.tar.gz"))
	got := FindOrphanDownloads(m, d.Downloads, 3)
	if len(got) != 1 || got[0].Source != "github.com/junegunn/fzf" || got[0].Version != "0.69.0" || got[0].File != "old.tar.gz" || got[0].Package() != "fzf" {
		t.Fatalf("FindOrphanDownloads = %+v", got)
	}
	if err := os.Remove(got[0].Path); err != nil {
		t.Fatal(err)
	}
	PruneEmptyDirs(d.Downloads)
	if exists(filepath.Join(d.Downloads, "github.com", "junegunn", "fzf", "0.69.0")) || !exists(filepath.Join(d.Downloads, "github.com", "junegunn", "fzf", "0.70.0")) {
		t.Error("PruneEmptyDirs took the wrong dirs")
	}

	gm := manifest.New()
	gm.Repos["go"] = "go.dev/dl"
	gm.Extracts["go"] = manifest.Entry{Version: "1.27.2"}
	root := filepath.Join(t.TempDir(), "download")
	touch(t, filepath.Join(root, "go.dev", "dl", "1.27.2", "go1.27.2.tar.gz"))
	touch(t, filepath.Join(root, "go.dev", "dl", "1.26.9", "go1.26.9.tar.gz"))
	two := FindOrphanDownloads(gm, root, 2)
	if len(two) != 1 || two[0].Version != "1.26.9" || two[0].Source != "go.dev/dl" {
		t.Fatalf("FindOrphanDownloads at depth 2 = %+v", two)
	}
	PruneDownloadVersion(root, "go.dev/dl", "1.26.9")
	PruneDownloadVersion(root, "go.dev/dl", "1.27.2")
	if exists(filepath.Join(root, "go.dev")) || !exists(root) {
		t.Error("PruneDownloadVersion left empty parents, or took the downloads dir")
	}
}

func TestOrphanExtractsInsideAnInstallWithALatestPointer(t *testing.T) {
	m, d := state(t)
	key := filepath.Join(d.Extracts, "fzf")
	for _, p := range []string{"0.69.0/fzf", "0.70.0/fzf", ".0.71.0.new/fzf", "loose"} {
		touch(t, filepath.Join(key, p))
	}
	if err := pointer.Point(filepath.Join(key, layout.Latest), filepath.Join(key, "0.70.0")); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range FindOrphanExtracts(m, d.Extracts) {
		if o.Key == "fzf" && o.Leftover {
			got = append(got, filepath.Base(o.Path))
		}
	}
	slices.Sort(got)
	if want := []string{".0.71.0.new", "0.69.0", "bin", "loose"}; !slices.Equal(got, want) {
		t.Errorf("orphans in fzf = %v, want %v", got, want)
	}
}

func TestBrokenWhenTheLatestPointerNamesNothing(t *testing.T) {
	m, d := state(t)
	key := filepath.Join(d.Extracts, "fzf")
	if err := pointer.Point(filepath.Join(key, layout.Latest), filepath.Join(key, "0.70.0")); err != nil {
		t.Fatal(err)
	}
	var missing bool
	for _, b := range FindBroken(m, d) {
		missing = missing || (b.Key == "fzf" && b.Issue == "missing extract")
	}
	if !missing {
		t.Error("a latest pointer to no version read as an install")
	}
}

func TestAnInstallWithNoLatestPointerIsWholeWhileItsDirIsThere(t *testing.T) {
	m, d := state(t)
	for _, b := range FindBroken(m, d) {
		if b.Issue == "missing extract" {
			t.Errorf("%s read as broken", b.Key)
		}
	}
	for _, o := range FindOrphanExtracts(m, d.Extracts) {
		t.Errorf("%s read as an orphan", o.Path)
	}
}

func TestPruneInstallKeepsOnlyTheLatestPointerAndItsVersion(t *testing.T) {
	_, d := state(t)
	key := filepath.Join(d.Extracts, "fzf")
	for _, p := range []string{"0.69.0/fzf", "0.70.0/fzf", "ripgrep-15.2.0/rg", "loose"} {
		touch(t, filepath.Join(key, p))
	}
	if stuck := PruneInstall(key); len(stuck) != 0 || !exists(filepath.Join(key, "loose")) {
		t.Fatalf("an install with no pointer was pruned: %v", stuck)
	}
	if err := pointer.Point(filepath.Join(key, layout.Latest), filepath.Join(key, "0.70.0")); err != nil {
		t.Fatal(err)
	}
	if stuck := PruneInstall(key); len(stuck) != 0 {
		t.Fatalf("stuck: %v", stuck)
	}
	entries, err := os.ReadDir(key)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if !slices.Equal(names, []string{"0.70.0", "latest"}) {
		t.Errorf("after the prune: %v", names)
	}
}
