// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/netsyinit"
	"github.com/spf13/cobra"
)

var hooksNetsyInitClusterConfig, hooksNetsyInitOutput, hooksNetsyInitTemplate, hooksNetsyInitValuesFile string

func newHooksNetsyInitCmd(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "netsy-init",
		Short: "Generate an initial Netsy snapshot file from a template",
		Long:  `Downloads or reads a Netsy snapshot template, interpolates cluster-specific platform component values from a Podplane cluster config, and writes the resulting .netsy snapshot file to stdout or --output file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var output *os.File
			closeOutput := false
			if hooksNetsyInitOutput == "" || hooksNetsyInitOutput == "-" {
				output = os.Stdout
			} else {
				output, err = os.Create(hooksNetsyInitOutput)
				if err != nil {
					return fmt.Errorf("create Netsy snapshot output %s: %w", hooksNetsyInitOutput, err)
				}
				closeOutput = true
			}
			err = netsyinit.WriteSnapshot(output, netsyinit.SnapshotOptions{
				ClusterConfigPath: hooksNetsyInitClusterConfig,
				Template:          hooksNetsyInitTemplate,
				DepsBaseURL:       c.DepsBaseURL(),
				ValuesFile:        hooksNetsyInitValuesFile,
			})
			if closeOutput {
				closeErr := output.Close()
				if closeErr != nil && err == nil {
					return closeErr
				}
			}
			return err
		},
	}
	cmd.Flags().StringVarP(&hooksNetsyInitClusterConfig, "cluster-config", "f", "podplane.cluster.jsonc", "Path to the cluster config file")
	cmd.Flags().StringVarP(&hooksNetsyInitOutput, "output", "o", "", "Path to write the generated Netsy snapshot file (defaults to stdout)")
	cmd.Flags().StringVar(&hooksNetsyInitTemplate, "template", "", "Path or HTTP(S) URL for the Netsy snapshot template (defaults to <deps-base-url>/netsy/recommended.netsy)")
	cmd.Flags().StringVar(&hooksNetsyInitValuesFile, "values", "", "Path to a YAML/JSON values file to merge over derived platform-components values")
	return cmd
}
