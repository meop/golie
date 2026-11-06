package sig

import (
	"bytes"
	"os"
	"testing"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// a real release's manifest and signature, as wedge 0.5.4's pipeline made them
func TestAReleaseManifestVerifiesWithTheReleaseKey(t *testing.T) {
	sums, signature := read(t, "wedge-0.5.4.SHA256SUMS"), read(t, "wedge-0.5.4.SHA256SUMS.sig")
	if err := Verify(ReleaseKey, sums, signature); err != nil {
		t.Fatal(err)
	}
}

func TestAChangedManifestDoesNotVerify(t *testing.T) {
	sums, signature := read(t, "wedge-0.5.4.SHA256SUMS"), read(t, "wedge-0.5.4.SHA256SUMS.sig")
	changed := bytes.Replace(sums, []byte("linux-x86_64"), []byte("linux-x86_65"), 1)
	if err := Verify(ReleaseKey, changed, signature); err == nil {
		t.Fatal("a manifest changed after it was signed verified")
	}
	if err := Verify(ReleaseKey, sums, signature[:len(signature)-1]); err == nil {
		t.Fatal("a cut signature verified")
	}
}
