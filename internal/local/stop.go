// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"

	"github.com/fatih/color"
)

// Stop stops the local cluster VM and supporting processes
func (m *Local) Stop() error {
	// Stop the VM
	fmt.Println("Stopping VM...")
	if err := m.vm.Stop(); err != nil {
		return err
	}
	if err := removeState(m.runtimeDir, m.clusterID); err != nil {
		return err
	}
	color.Green("✅ VM stopped successfully")

	// Stop the local server background process if this is the last running VM
	if err := m.ServerCleanup(); err != nil {
		return fmt.Errorf("failed to stop background server for local clusters: %w", err)
	}

	color.Green("✅ Local server stopped successfully")

	return nil
}
