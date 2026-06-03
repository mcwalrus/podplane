// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/local"
	"github.com/spf13/cobra"
)

var localConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Attach to the local cluster VM serial console",
	Long:  `Attach to the local cluster VM serial console for boot and login debugging. Press Ctrl-] to detach from the console without stopping the VM.`,
}

// newLocalConsoleCmd creates the `local console` command.
func newLocalConsoleCmd(c *config.Config) *cobra.Command {
	localConsoleCmd.RunE = func(cmd *cobra.Command, args []string) error {
		manager := local.NewManager(c, localClusterID)
		if err := manager.Console(); err != nil {
			return fmt.Errorf("failed to attach console: %w", err)
		}
		return nil
	}

	return localConsoleCmd
}
