// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadSecretValueFromFile verifies --file reads the exact file bytes.
func TestReadSecretValueFromFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "secret-value")
	if err := os.WriteFile(path, []byte("secret\nvalue"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	value, err := readSecretValue("create", "api-key", false, path)
	if err != nil {
		t.Fatalf("readSecretValue() error = %v", err)
	}
	if got, want := string(value), "secret\nvalue"; got != want {
		t.Fatalf("readSecretValue() = %q, want %q", got, want)
	}
}

// TestReadSecretValueRejectsFileAndStdin verifies --file and --stdin cannot both be set.
func TestReadSecretValueRejectsFileAndStdin(t *testing.T) {
	t.Parallel()
	_, err := readSecretValue("create", "api-key", true, "secret.txt")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("readSecretValue() error = %v, want mutual exclusion error", err)
	}
}
