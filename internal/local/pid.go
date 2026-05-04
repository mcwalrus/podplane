// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/podplane/podplane/internal/pid"
)

// ServerPIDFile loads (or initializes) the PID file used to track the
// background `podplane local server` process. runtimeDir is the directory in
// which the file is stored (typically <Config.RuntimeDirectory()>).
func ServerPIDFile(runtimeDir string) (pid.PIDFile, error) {
	return pid.Init(pid.PIDOptions{
		Directory: runtimeDir,
		Filename:  "local-server.json",
	})
}
