// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"

	"github.com/fatih/color"
)

// Status checks and displays the status of the local cluster VM
func (m *Local) Status() error {
	// Check if VM exists
	exists, err := m.vm.Exists()
	if err != nil {
		return err
	}

	if !exists {
		color.Red("VM Status: Not created")
	} else {
		// Check VM status
		running, err := m.vm.Running()
		if err != nil {
			return err
		}

		// Display status
		if running {
			if vmPID, err := m.vm.PID(); err == nil && vmPID != 0 {
				color.Green(fmt.Sprintf("VM Status: Running (PID %d)", vmPID))
			} else {
				color.Green("VM Status: Running")
			}
		} else {
			color.Yellow("VM Status: Stopped")
		}
	}

	// Check local server status
	pidFile, err := ServerPIDFile(m.runtimeDir)
	if err != nil {
		color.Red("Local Server: Unknown (failed to load PID file)")
		return nil
	}

	isRunning, err := pidFile.IsRunning()
	if err != nil || !isRunning {
		color.Red("Local Server: Not running")
		color.Yellow(fmt.Sprintf("Local Server Log: %s", ServerLogPath(m.runtimeDir)))
		return nil
	}

	host := pidFile.GetData("host")
	httpPort := pidFile.GetData("http_port")
	httpsPort := pidFile.GetData("https_port")
	color.Green(fmt.Sprintf("Local Server: Running (PID %d) at HTTP %s:%s and HTTPS %s:%s", pidFile.PID(), host, httpPort, host, httpsPort))
	if logPath := pidFile.GetData("log_file"); logPath != "" {
		color.Green(fmt.Sprintf("Local Server Log: %s", logPath))
	} else {
		color.Green(fmt.Sprintf("Local Server Log: %s", ServerLogPath(m.runtimeDir)))
	}

	return nil
}
