// Podplane <https://podplane.dev>
// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Nadrama Pty Ltd

package main

import (
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/cmd"
	"github.com/podplane/podplane/internal/config"
)

func main() {
	c, err := config.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing config:", err)
		os.Exit(1)
	}
	root := cmd.NewRootCmd(c)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
