// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package pid

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v4/process"
)

// IsRunning checks if the PID file is running
func (p *PIDFile) IsRunning() (bool, error) {
	// if the PID is 0, the process is not running
	if p.PID() == 0 {
		return false, nil
	}
	// find the process by PID
	isRunning, err := process.PidExists(int32(p.PID()))
	if err != nil {
		return false, fmt.Errorf("failed to find process: %w", err)
	}
	// If the process is not running, remove the PID file and then the PID
	if !isRunning {
		if err := os.Remove(p.FilePath()); err != nil {
			panic(fmt.Errorf("failed to remove PID file %s: %w. please manually remove it", p.FilePath(), err))
		}
		p.pid = 0
	}
	return isRunning, nil
}
