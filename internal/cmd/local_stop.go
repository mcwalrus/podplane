// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/clusterauth"
	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/local"
	"github.com/podplane/podplane/internal/vm"
	"github.com/spf13/cobra"
)

var localStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a local cluster VM",
	Long:  `Stop a running local cluster VM`,
}

func init() {
	localStopCmd.Flags().BoolVar(&localStopRemove, "rm", false, "Remove (delete) the cluster after stopping")
}

var localStopRemove bool

func newLocalStopCmd(c *config.Config) *cobra.Command {
	localStopCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create local cluster manager and stop the VM
		manager := local.NewManager(c, localClusterID)
		if err := manager.Stop(); err != nil {
			if localStopRemove && errors.Is(err, vm.ErrNotRunning) {
				fmt.Fprintln(os.Stdout, "VM is already stopped; continuing with removal...")
				if err := manager.ServerCleanup(); err != nil {
					return fmt.Errorf("failed to stop background server for local clusters: %w", err)
				}
			} else {
				return fmt.Errorf("failed to stop: %w", err)
			}
		}

		// If --rm flag is set, also delete the cluster
		if localStopRemove {
			if err := clusterauth.Logout(c, os.Stdout, localClusterID, true); err != nil {
				return fmt.Errorf("failed to clear local cluster credentials: %w", err)
			}
			if err := manager.Delete(); err != nil {
				return fmt.Errorf("failed to delete after stop: %w", err)
			}
		}

		return nil
	}

	return localStopCmd
}
