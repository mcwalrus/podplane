// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package qemu

import (
	"fmt"
)

// Running checks if a Qemu VM is currently running, first by
// loading our PID file, then by verifying the process is running.
func (m *Qemu) Running() (bool, error) {
	// check VM exists first
	exists, err := m.Exists()
	if !exists || err != nil {
		return false, err
	}

	// Get PID file
	pidFile, err := m.VMPIDFile()
	if err != nil {
		return false, fmt.Errorf("failed to get VM PID file: %w", err)
	}

	// Check VM if is already running
	return pidFile.IsRunning()
}

// PID returns the process ID of the running VM, or 0 if not running.
func (m *Qemu) PID() (int, error) {
	pidFile, err := m.VMPIDFile()
	if err != nil {
		return 0, fmt.Errorf("failed to get VM PID file: %w", err)
	}
	isRunning, err := pidFile.IsRunning()
	if err != nil || !isRunning {
		return 0, err
	}
	return pidFile.PID(), nil
}
