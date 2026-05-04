// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

// Console attaches to the running local cluster VM's serial console.
func (m *Local) Console() error {
	return m.vm.Console()
}
