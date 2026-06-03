// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/deploy"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Tail logs for a deployed app",
	Example: "  # Tail logs for the hello app\n" +
		"  podplane logs hello",
	Args: cobra.ExactArgs(1),
}

var (
	logsNamespace  string
	logsContext    string
	logsKubeconfig string
)

func init() {
	logsCmd.Flags().StringVarP(&logsNamespace, "namespace", "n", "", "Kubernetes namespace the app was deployed into")
	logsCmd.Flags().StringVar(&logsContext, "context", "", "The name of the kubeconfig context to use")
	logsCmd.Flags().StringVar(&logsKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
}

func newLogsCmd(c *config.Config) *cobra.Command {
	_ = c
	logsCmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return deploy.Logs(deploy.LogsOptions{
			Name:       args[0],
			Namespace:  logsNamespace,
			Context:    logsContext,
			Kubeconfig: logsKubeconfig,
		})
	}
	return logsCmd
}
