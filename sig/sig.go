// Package sig checks a release's checksum manifest against the detached
// OpenPGP signature its pipeline publishes beside it.
package sig

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// ReleaseKey is the public key every meop release pipeline signs its
// SHA256SUMS with: wedge's, delish's, ghpm's and goup's. It is the one thing a
// tool must already hold to trust what it downloads, so it is built in, never
// fetched.
//
//go:embed release.asc
var ReleaseKey []byte

// Verify reports whether signature is a valid detached signature over signed
// by key, an armored OpenPGP public key.
func Verify(key, signed, signature []byte) error {
	ring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(key))
	if err != nil {
		return fmt.Errorf("reading the release key: %w", err)
	}
	if _, err := openpgp.CheckDetachedSignature(ring, bytes.NewReader(signed), bytes.NewReader(signature), nil); err != nil {
		return fmt.Errorf("signature does not verify: %w", err)
	}
	return nil
}
