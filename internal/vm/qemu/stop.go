// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package qemu

import (
	"fmt"

	"github.com/podplane/podplane/internal/vm"
)

// Stop stops a qemu VM
func (m *Qemu) Stop() error {
	// Check if VM is running
	running, err := m.Running()
	if err != nil {
		return err
	}
	if !running {
		return vm.ErrNotRunning
	}

	// Get PID file
	pidFile, err := m.VMPIDFile()
	if err != nil {
		return fmt.Errorf("failed to get VM PID file: %w", err)
	}

	// Stop the VM
	err = pidFile.Kill()
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	return nil
}
