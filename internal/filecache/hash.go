// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package filecache

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
)

// newHash returns a hash.Hash for the given algorithm name.
// Supported algorithms: "sha256", "sha512".
func newHash(algo string) (hash.Hash, error) {
	switch algo {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", algo)
	}
}
