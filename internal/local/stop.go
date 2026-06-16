// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fatih/color"

	"github.com/podplane/podplane/internal/vm"
)

// Stop stops the local cluster VM and supporting processes
func (m *Local) Stop() error {
	// Best-effort: let containerd stop cleanly before QEMU is terminated, so
	// image pulls/unpacks are less likely to be interrupted mid-write.
	if state, err := readState(m.runtimeDir, m.clusterID); err == nil && state.Ports.SSH != 0 {
		if privateKeyPath, err := SSHPrivateKeyPath(m.dataDir); err == nil {
			fmt.Println("Stopping containerd...")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if _, err := m.vm.Shell(ctx, "sudo systemctl stop containerd", state.Ports.SSH, privateKeyPath, vm.ShellOptions{Timeout: 10 * time.Second}); err == nil {
				color.Green("✓ containerd stopped successfully")
			}
			cancel()
		}
	}

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
