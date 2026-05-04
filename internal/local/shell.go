// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"fmt"

	"github.com/podplane/podplane/internal/vm"
)

// Shell opens a shell into the running local cluster VM
// If command is provided, executes that command instead
// of opening interactive shell.
func (m *Local) Shell(command string) error {
	state, err := readState(m.runtimeDir, m.clusterID)
	if err != nil {
		return err
	}
	sshPort := state.Ports.SSH
	if sshPort == 0 {
		return fmt.Errorf("state is missing ssh port")
	}
	_, err = m.vm.Shell(context.Background(), command, sshPort, vm.ShellOptions{})
	return err
}
