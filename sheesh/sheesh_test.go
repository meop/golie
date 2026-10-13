package sheesh

import (
	"strings"
	"testing"
)

func TestTool(t *testing.T) {
	for _, tc := range []struct {
		goos, goarch, want string
		files              []string
	}{
		{"linux", "amd64", "sheesh-0.1.1-linux-x86_64.tar.gz", []string{"kebab", "sheesh"}},
		{"linux", "arm64", "sheesh-0.1.1-linux-aarch64.tar.gz", []string{"kebab", "sheesh"}},
		{"darwin", "arm64", "sheesh-0.1.1-darwin-aarch64.tar.gz", []string{"kebab", "sheesh"}},
		{"darwin", "amd64", "sheesh-0.1.1-darwin-x86_64.tar.gz", []string{"kebab", "sheesh"}},
		{"windows", "amd64", "sheesh-0.1.1-windows-x86_64.zip", []string{"kebab.exe", "sheesh.exe", "sheesh-gui.exe"}},
		{"windows", "arm64", "sheesh-0.1.1-windows-aarch64.zip", []string{"kebab.exe", "sheesh.exe", "sheesh-gui.exe"}},
	} {
		tool, ok := Tool(tc.goos, tc.goarch)
		if !ok || !strings.HasSuffix(tool.Asset.URL, "/v0.1.1/"+tc.want) || len(tool.Asset.SHA256) != 64 {
			t.Errorf("Tool(%s, %s) = %+v, %v", tc.goos, tc.goarch, tool.Asset.URL, ok)
		}
		if len(tool.Asset.Files) != len(tc.files) {
			t.Errorf("Tool(%s, %s) pins %d files, want %v", tc.goos, tc.goarch, len(tool.Asset.Files), tc.files)
		}
		for _, f := range tc.files {
			if len(tool.Asset.Files[f]) != 64 {
				t.Errorf("Tool(%s, %s) has no digest for %s", tc.goos, tc.goarch, f)
			}
		}
	}
	if _, ok := Tool("freebsd", "amd64"); ok {
		t.Error("sheesh ships nothing for freebsd, and Tool named something")
	}
}
