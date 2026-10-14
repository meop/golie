package manifest

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadStartsEmpty(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), "manifest.json"))
	if err != nil || m.Repos == nil || m.Extracts == nil || len(m.Extracts) != 0 {
		t.Fatalf("fresh manifest = %+v, %v", m, err)
	}
}

// the file is the contract two tools share, so its field names are pinned here
func TestSaveWritesTheContract(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	path := filepath.Join(dir, "manifest.json")
	m := New()
	m.Repos["go"] = "go.dev/dl"
	m.Extracts["go@1.26"] = Entry{Pin: PinMinor, Version: "1.26.9", Assets: []string{"a.tar.gz"}, Bin: map[string]string{"go@1.26": "bin/go"}}
	if err := Save(m, path); err != nil {
		t.Fatal(err)
	}
	want := `{
  "repo": {
    "go": "go.dev/dl"
  },
  "extract": {
    "go@1.26": {
      "pin": "minor",
      "version": "1.26.9",
      "assets": [
        "a.tar.gz"
      ],
      "bin": {
        "go@1.26": "bin/go"
      }
    }
  }
}
`
	if got, _ := os.ReadFile(path); string(got) != want {
		t.Errorf("saved\n%s\nwant\n%s", got, want)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("a save left %d files beside the manifest", len(entries)-1)
	}
	back, err := Load(path)
	if err != nil || back.Extracts["go@1.26"].Bin["go@1.26"] != "bin/go" {
		t.Errorf("loaded %+v, %v", back, err)
	}
}

func TestLoadRejectsWhatIsNotAManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("a truncated manifest loaded")
	}
}

// a package's source is forgotten with its last install, not its first
func TestRemoveExtract(t *testing.T) {
	m := New()
	m.Repos["fzf"] = "github.com/junegunn/fzf"
	m.Extracts["fzf"] = Entry{}
	m.Extracts["fzf@0.70"] = Entry{}
	m.RemoveExtract("fzf")
	if _, ok := m.Repos["fzf"]; !ok {
		t.Fatal("the source went while fzf@0.70 was still installed")
	}
	m.RemoveExtract("fzf@0.70")
	if len(m.Repos) != 0 || len(m.Extracts) != 0 {
		t.Fatalf("left %+v", m)
	}
}

func TestFindBySourceAndKeys(t *testing.T) {
	m := New()
	m.Repos["rg"] = "github.com/BurntSushi/ripgrep"
	m.Extracts["rg"] = Entry{}
	m.Extracts["fzf@14"] = Entry{}
	if name, ok := m.FindBySource("github.com/BurntSushi/ripgrep"); !ok || name != "rg" {
		t.Errorf("FindBySource = %q, %v", name, ok)
	}
	if _, ok := m.FindBySource("github.com/nobody/nothing"); ok {
		t.Error("found a source nothing is registered with")
	}
	if got := m.Keys(); !slices.Equal(got, []string{"fzf@14", "rg"}) {
		t.Errorf("Keys = %v", got)
	}
}

func TestDiscovered(t *testing.T) {
	e := Entry{Bin: map[string]string{"a": "bin/a"}, BinDeclined: []string{"bin/b"}, Font: map[string]string{"f": "f.ttf"}, FontDeclined: []string{"g.ttf"}}
	if got := e.DiscoveredBins(); !slices.Equal(got, []string{"bin/a", "bin/b"}) {
		t.Errorf("DiscoveredBins = %v", got)
	}
	if got := e.DiscoveredFonts(); !slices.Equal(got, []string{"f.ttf", "g.ttf"}) {
		t.Errorf("DiscoveredFonts = %v", got)
	}
}

func TestKeys(t *testing.T) {
	for _, tc := range []struct {
		name, version, key, pin string
	}{
		{"fzf", "", "fzf", PinLatest},
		{"fzf", "14", "fzf@14", PinMajor},
		{"fzf", "v14.1", "fzf@14.1", PinMinor},
		{"go", "1.26.4", "go@1.26.4", PinFixed},
		// a pin that is not numbers all the way names one release
		{"go", "1.28rc1", "go@1.28rc1", PinFixed},
	} {
		key := Key(tc.name, tc.version)
		if key != tc.key {
			t.Errorf("Key(%q, %q) = %q, want %q", tc.name, tc.version, key, tc.key)
		}
		if got := Pin(tc.version); got != tc.pin {
			t.Errorf("Pin(%q) = %q, want %q", tc.version, got, tc.pin)
		}
		name, version, pinned := ParseKey(key)
		if name != tc.name || pinned != (tc.version != "") || (pinned && "v"+version != tc.version && version != tc.version) {
			t.Errorf("ParseKey(%q) = %q, %q, %v", key, name, version, pinned)
		}
	}
}

func TestShimNameAndBinPath(t *testing.T) {
	for _, tc := range []struct{ key, bin, want string }{
		{"go", "go", "go"},
		{"go", "go.exe", "go.exe"},
		{"go@1.26", "gofmt", "gofmt@1.26"},
		{"go@1.26", "gofmt.exe", "gofmt@1.26.exe"},
	} {
		if got := ShimName(tc.key, tc.bin); got != tc.want {
			t.Errorf("ShimName(%q, %q) = %q, want %q", tc.key, tc.bin, got, tc.want)
		}
	}
	if dir, name := ParseBinPath("bin/go"); dir != "bin" || name != "go" {
		t.Errorf("ParseBinPath(bin/go) = %q, %q", dir, name)
	}
	if dir, name := ParseBinPath("fzf"); dir != "" || name != "fzf" {
		t.Errorf("ParseBinPath(fzf) = %q, %q", dir, name)
	}
}
