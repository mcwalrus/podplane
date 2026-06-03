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

var localStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of a local cluster VM",
	Long:  `Check the local cluster VM status.`,
}

func newLocalStatusCmd(c *config.Config) *cobra.Command {
	localStatusCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create local cluster manager and check the VM status
		manager := local.NewManager(c, localClusterID)
		if err := manager.Status(); err != nil {
			return fmt.Errorf("failed to check status: %w", err)
		}
		return nil
	}

	return localStatusCmd
}
