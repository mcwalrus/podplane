// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package qemu

import (
	"fmt"
	"os"
)

// Exists checks if a Qemu VM image exists on disk
func (m *Qemu) Exists() (bool, error) {
	// Path to VM image
	vmImage := m.VMImagePath()

	// Check if the VM image file exists
	stat, err := os.Stat(vmImage)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check VM image file: %w", err)
	} else if stat.IsDir() {
		return false, fmt.Errorf("VM image path is a directory: %s", vmImage)
	}
	return true, nil
}
