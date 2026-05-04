// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/clusterconfig"
	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/deps"
	"github.com/podplane/podplane/internal/kubectl"
	"github.com/podplane/podplane/internal/local"
	"github.com/podplane/podplane/internal/oidcserver"
	"github.com/podplane/podplane/internal/tui"
	"github.com/podplane/podplane/internal/vm"
	"github.com/spf13/cobra"
)

var localStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a local cluster VM",
	Long:  `Create and start a local cluster VM`,
}

func init() {
	localStartCmd.Flags().StringVarP(&localStartCPUs, "cpus", "c", "2", "CPU cores to allocate to the VM (default 2)")
	localStartCmd.Flags().StringVarP(&localStartMemory, "memory", "m", "4G", "Memory to allocate to the VM (default 4G)")
	localStartCmd.Flags().BoolVar(&localStartConsole, "console", false, "Attach to the VM serial console after startup")
	localStartCmd.Flags().BoolVar(&localStartFollow, "follow", false, "Stream cloud-init user-data logs while waiting for startup")
}

var (
	localStartCPUs    string
	localStartMemory  string
	localStartConsole bool
	localStartFollow  bool
)

// newLocalStartCmd creates the `local start` command. After the VM is up it
// configures kubectl (cluster + credentials + context) so that the very next
// `kubectl` command invokes the `podplane hooks kubectl-auth` plugin,
// which performs the OIDC login.
func newLocalStartCmd(c *config.Config) *cobra.Command {
	localStartCmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// Create local cluster manager and start the VM
		manager := local.NewManager(c, localClusterID)
		stashPath, err := manager.Start(local.StartOptions{
			CPUs:               localStartCPUs,
			Memory:             localStartMemory,
			StreamUserdataLogs: localStartFollow,
			RunDownloadProgress: func(run func(progress func(deps.DownloadEvent)) error) error {
				return tui.RunDownloadProgress("Downloading Podplane dependencies", run)
			},
		})
		if err != nil {
			if errors.Is(err, vm.ErrAlreadyRunning) {
				return fmt.Errorf("VM is already running — use `podplane local stop` first, or `podplane local status` to check")
			}
			return fmt.Errorf("failed to start: %w", err)
		}
		if stashPath == "" {
			return nil
		}
		cluster, err := clusterconfig.Load(stashPath)
		if err != nil {
			return fmt.Errorf("load local cluster config: %w", err)
		}
		// Local OIDC always issues tokens with sub == oidcserver.LocalSub, so we
		// can configure kubectl now without first performing a login.
		if err := kubectl.ConfigureClusterAccess(os.Stdout, cluster.Cluster.ID, cluster.ResolvedKubernetesAPIURL(), oidcserver.LocalSub, "", true); err != nil {
			return fmt.Errorf("configure kubectl: %w", err)
		}
		fmt.Printf("✅ kubectl configured for local cluster using %q context\n", kubectl.ContextKey(cluster.Cluster.ID, true))
		if ingressURL, err := manager.LocalIngressURL(); err == nil {
			fmt.Printf("Local ingress proxy: %s\n", ingressURL)
		}
		if localStartConsole {
			if err := manager.Console(); err != nil {
				return fmt.Errorf("attach console: %w", err)
			}
		}
		return nil
	}

	return localStartCmd
}
