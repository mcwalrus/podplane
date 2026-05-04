// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/podplane/podplane/internal/config"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Hooks for integration with tools like kubectl",
	Long:  `Hooks allow other tools to integrate with the Podplane CLI, such as kubectl using 'podplane' as an auth exec command`,
}

// newHooksCmd creates the `hooks` command and its subcommands.
func newHooksCmd(c *config.Config) *cobra.Command {
	hooksCmd.Run = func(cmd *cobra.Command, args []string) {
		// If no subcommand provided, show help
		_ = cmd.Help()
	}

	hooksCmd.AddCommand(newHooksKubectlAuthCmd(c))
	hooksCmd.AddCommand(newHooksNetsyInitCmd(c))

	return hooksCmd
}
