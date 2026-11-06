// Package testsign makes a throwaway OpenPGP key for tests that need a
// release signed by a key other than the real one.
package testsign

import (
	"bytes"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// New returns a new key's armored public half, and a function that signs with
// it as a release pipeline does: a binary detached signature.
func New(t *testing.T) (key []byte, sign func(data []byte) []byte) {
	t.Helper()
	e, err := openpgp.NewEntity("test", "", "test@example.com", &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA})
	if err != nil {
		t.Fatal(err)
	}
	var pub bytes.Buffer
	w, err := armor.Encode(&pub, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Serialize(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return pub.Bytes(), func(data []byte) []byte {
		var out bytes.Buffer
		if err := openpgp.DetachSign(&out, e, bytes.NewReader(data), nil); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
}
