// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/podplane/podplane/internal/buildvars"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version of Podplane CLI",
	Long:  `Prints the current installed Podplane CLI version information`,
}

// newVersionCmd creates the `version` command.
func newVersionCmd() *cobra.Command {
	versionCmd.Run = func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s (%s)\n", buildvars.BuildVersion(), buildvars.CommitHash())
	}
	return versionCmd
}
