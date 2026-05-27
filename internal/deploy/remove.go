// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deploy

// RemoveOptions controls removal of a previously deployed app.
type RemoveOptions struct {
	Name       string
	Namespace  string
	Context    string
	Kubeconfig string
}

// Remove invokes `helm uninstall` for the named release.
func Remove(opts RemoveOptions) error {
	args := []string{"uninstall", opts.Name}
	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}
	if opts.Context != "" {
		args = append(args, "--kube-context", opts.Context)
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig", opts.Kubeconfig)
	}
	return runHelm(args)
}
