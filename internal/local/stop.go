// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"errors"
	"fmt"

	"github.com/fatih/color"

	"github.com/podplane/podplane/internal/vm"
)

// Stop stops the local cluster VM and supporting processes
func (m *Local) Stop() error {
	// Stop the VM
	fmt.Println("Stopping VM...")
	if err := m.vm.Stop(); err != nil {
		if !errors.Is(err, vm.ErrNotRunning) {
			return err
		}
		fmt.Println("VM is already stopped")
	} else {
		color.Green("✓ VM stopped successfully")
	}
	if err := removeState(m.runtimeDir, m.clusterID); err != nil {
		return err
	}

	// Stop the local server background process if this is the last running VM
	if err := m.ServerCleanup(); err != nil {
		return fmt.Errorf("failed to stop background server for local clusters: %w", err)
	}

	color.Green("✓ Local server stopped successfully")

	return nil
}
