// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// PubkeyForSshAuthorizedKey loads a key for the ssh authorized_keys file
// from your ~/.ssh/id_*.pub files, preferring ed25519, then ecdsa, then rsa.
func PubkeyForSshAuthorizedKey() (string, error) {
	// Get user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Path to .ssh directory
	sshDir := filepath.Join(home, ".ssh")

	// Check if .ssh directory exists
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		// .ssh directory doesn't exist, return empty string without error
		// likely to happen on Windows!
		return "", nil
	}

	// Try to find public keys in priority order
	keyNames := []string{"id_ed25519.pub", "id_ecdsa.pub", "id_rsa.pub"}

	// Check priority keys first
	for _, keyName := range keyNames {
		keyPath := filepath.Join(sshDir, keyName)
		if keyData, err := os.ReadFile(keyPath); err == nil {
			return string(bytes.TrimSpace(keyData)), nil
		}
	}

	// If no priority keys found, try any other id_*.pub files
	pattern := filepath.Join(sshDir, "id_*.pub")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to find public keys: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no SSH public keys found in %s", sshDir)
	}

	// Read the first available key
	keyData, err := os.ReadFile(matches[0])
	if err != nil {
		return "", fmt.Errorf("failed to read key file %s: %w", matches[0], err)
	}

	return string(bytes.TrimSpace(keyData)), nil
}
