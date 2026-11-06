package wedge

import (
	"slices"
	"testing"
)

func TestRelease(t *testing.T) {
	for _, tc := range []struct {
		goos, goarch, want string
		files              []string
	}{
		{"linux", "amd64", "wedge-0.5.4-linux-x86_64.tar.gz", []string{"mallet", "wedge"}},
		{"linux", "arm64", "wedge-0.5.4-linux-aarch64.tar.gz", []string{"mallet", "wedge"}},
		{"darwin", "arm64", "wedge-0.5.4-darwin-aarch64.tar.gz", []string{"mallet", "wedge"}},
		{"windows", "amd64", "wedge-0.5.4-windows-x86_64.zip", []string{"mallet.exe", "wedge.exe", "wedge-gui.exe"}},
		{"windows", "arm64", "wedge-0.5.4-windows-aarch64.zip", []string{"mallet.exe", "wedge.exe", "wedge-gui.exe"}},
	} {
		r, ok := Release(tc.goos, tc.goarch)
		if !ok {
			t.Errorf("Release(%s, %s) names nothing", tc.goos, tc.goarch)
			continue
		}
		if got, _ := r.Archive("0.5.4"); got != tc.want {
			t.Errorf("Release(%s, %s) archive = %s, want %s", tc.goos, tc.goarch, got, tc.want)
		}
		if !slices.Equal(r.Files, tc.files) || r.Key == nil {
			t.Errorf("Release(%s, %s) = %v, key %v", tc.goos, tc.goarch, r.Files, r.Key != nil)
		}
	}
	for _, p := range [][2]string{{"darwin", "amd64"}, {"freebsd", "amd64"}, {"linux", "386"}} {
		if _, ok := Release(p[0], p[1]); ok {
			t.Errorf("wedge ships nothing for %s/%s, and Release named something", p[0], p[1])
		}
	}
}
