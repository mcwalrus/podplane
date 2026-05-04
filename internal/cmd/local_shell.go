// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/local"
	"github.com/spf13/cobra"
)

var localShellCmd = &cobra.Command{
	Use:   "shell [command]",
	Short: "Open a shell into the local cluster VM or run a command via ssh",
	Long:  `Open a shell session into the local cluster VM, or if a command is provided it will run the command on the VM via SSH and exit immediately.`,
	Args:  cobra.MaximumNArgs(1),
}

// newLocalShellCmd creates the `local shell` command
func newLocalShellCmd(c *config.Config) *cobra.Command {
	localShellCmd.SilenceUsage = true
	localShellCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create local cluster manager and open a shell into the VM
		manager := local.NewManager(c, localClusterID)

		// Extract command from args if provided
		var command string
		if len(args) > 0 {
			command = args[0]
		}

		if err := manager.Shell(command); err != nil {
			return fmt.Errorf("failed to open shell: %w", err)
		}
		return nil
	}

	return localShellCmd
}
