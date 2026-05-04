// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/clusterauth"
	"github.com/podplane/podplane/internal/clusterconfig"
	"github.com/podplane/podplane/internal/config"
	"github.com/spf13/cobra"
)

var (
	logoutClusterConfig string
	logoutCluster       string
)

// newLogoutCmd creates the `podplane logout` command.
//
// Logout clears local state only — it does not revoke tokens at the issuer.
func newLogoutCmd(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached credentials for a Podplane cluster",
		Long: `Clear cached credentials for a Podplane cluster.

Removes both the metadata stored in the config file and the tokens stored in
the OS keyring for every (sub, cluster) pair matching the resolved cluster,
then removes the matching kubectl user, cluster, and context entries. Does not
revoke the tokens at the issuer.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterID := logoutCluster
			switch {
			case logoutClusterConfig != "" && logoutCluster != "":
				return fmt.Errorf("--cluster-config and --cluster are mutually exclusive")
			case logoutClusterConfig == "" && logoutCluster == "":
				return fmt.Errorf("one of -f/--cluster-config or --cluster is required")
			case logoutClusterConfig != "":
				cfg, err := clusterconfig.Load(logoutClusterConfig)
				if err != nil {
					return err
				}
				clusterID = cfg.Cluster.ID
			}
			return clusterauth.Logout(c, os.Stdout, clusterID, false)
		},
	}
	cmd.Flags().StringVarP(&logoutClusterConfig, "cluster-config", "f", "", "Path to a podplane.cluster.jsonc file")
	cmd.Flags().StringVar(&logoutCluster, "cluster", "", "Cluster ID")
	return cmd
}
