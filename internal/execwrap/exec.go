// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package execwrap

import (
	"log/slog"
	"os/exec"
	"strings"
)

// Command creates an exec.Command and logs the invocation at debug level.
func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)

	slog.Debug("executing command",
		"command", name,
		"args", arg,
		"line", "$ "+name+" "+strings.Join(arg, " "),
	)

	return cmd
}
