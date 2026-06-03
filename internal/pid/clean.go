// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package pid

import "os"

// Clean removes the PID file if it exists
func (p *PIDFile) Clean() error {
	pidFile := p.FilePath()
	if _, err := os.Stat(pidFile); err == nil {
		return os.Remove(pidFile)
	}
	return nil
}
