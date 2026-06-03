// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"os"

	"github.com/podplane/podplane/internal/execwrap"
)

// LogsOptions controls log streaming for a deployed app.
type LogsOptions struct {
	Name       string
	Namespace  string
	Context    string
	Kubeconfig string
}

// Logs invokes `kubectl logs --follow` for pods belonging to the named app.
func Logs(opts LogsOptions) error {
	cmd := execwrap.Command("kubectl", logsArgs(opts)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func logsArgs(opts LogsOptions) []string {
	args := []string{}
	if opts.Context != "" {
		args = append(args, "--context", opts.Context)
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig", opts.Kubeconfig)
	}
	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}
	args = append(args,
		"logs",
		"--follow",
		"--all-containers=true",
		"-l", "app.kubernetes.io/instance="+opts.Name,
	)
	return args
}
