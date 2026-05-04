// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import "fmt"

// Sync rsync's files into the running local cluster VM.
func (m *Local) Sync(from, to, chown string, excludes []string) error {
	state, err := readState(m.runtimeDir, m.clusterID)
	if err != nil {
		return err
	}
	sshPort := state.Ports.SSH
	if sshPort == 0 {
		return fmt.Errorf("state is missing ssh port")
	}
	return m.vm.Sync(from, to, chown, excludes, sshPort)
}
