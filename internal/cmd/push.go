// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/registryclient"
	"github.com/spf13/cobra"
)

var (
	pushContext    string
	pushKubeconfig string
)

// newPushCmd creates the `podplane push` command.
func newPushCmd(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <local-image> [<remote-image>]",
		Short: "Push a local image to the cluster registry",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			remote := ""
			if len(args) > 1 {
				remote = args[1]
			}
			ref, err := registryclient.Push(context.Background(), registryclient.Options{
				Config:     c,
				Source:     args[0],
				Remote:     remote,
				Context:    pushContext,
				Kubeconfig: pushKubeconfig,
				Stderr:     os.Stderr,
			})
			if err != nil {
				return err
			}
			fmt.Println(ref)
			return nil
		},
	}
	cmd.Flags().StringVar(&pushContext, "context", "", "kubeconfig context to use (default: current kubeconfig context)")
	cmd.Flags().StringVar(&pushKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	return cmd
}
