// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"os"

	"github.com/podplane/podplane/internal/execwrap"
)

// runHelm executes `helm` with the given args, streaming stdio to the current
// process.
func runHelm(args []string) error {
	cmd := execwrap.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
