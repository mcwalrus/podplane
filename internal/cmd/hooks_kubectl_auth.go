// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/podplane/podplane/internal/clusterauth"
	"github.com/podplane/podplane/internal/config"
	"github.com/spf13/cobra"
)

var (
	hooksKubectlAuthCluster string
	hooksKubectlAuthUser    string
)

// kubectlResponseStatus is a struct for the JSON response kubectl expects.
type kubectlResponseStatus struct {
	Error string `json:"error,omitempty"`
	Token string `json:"token,omitempty"`
}

// kubectlResponse is a struct for the JSON response kubectl expects.
type kubectlResponse struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Status     kubectlResponseStatus `json:"status"`
}

// kubectlReturn accepts either a token or an error and prints a JSON response
// to stdout from kubectlResponse. Always returns nil so cobra doesn't also
// print the error to stderr — kubectl reads the ExecCredential off our
// stdout and the error is already encoded into Status.Error there.
func kubectlReturn(stdout *os.File, token string, err error) error {
	resp := kubectlResponse{
		APIVersion: "client.authentication.k8s.io/v1beta1",
		Kind:       "ExecCredential",
	}
	if err != nil {
		resp.Status.Error = err.Error()
	} else {
		resp.Status.Token = token
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
	return nil
}

// newHooksKubectlAuthCmd implements the kubectl client-go ExecCredential
// plugin. kubectl invokes:
//
//	podplane hooks kubectl-auth --cluster <clusterID> --user <sub>
//
// Resolution order:
//
//  1. Cached id_token still valid -> emit it.
//  2. Refresh succeeds -> persist new tokens, emit id_token.
//  3. Otherwise perform a fresh login (headless against the local fake OIDC,
//     interactive browser-based against any remote production issuer — the
//     same way kubelogin does it), persist tokens, emit id_token.
func newHooksKubectlAuthCmd(c *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl-auth",
		Short: "kubectl exec auth plugin (issues an ExecCredential)",
		Long: `Implements the kubectl client-go ExecCredential interface.
Reads the cached OIDC tokens for the given (cluster, user) pair, refreshes
them if possible, otherwise performs a fresh login. Prints an ExecCredential
JSON document on stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hooksKubectlAuthCluster == "" {
				return kubectlReturn(os.Stdout, "", fmt.Errorf("--cluster is required"))
			}
			if hooksKubectlAuthUser == "" {
				return kubectlReturn(os.Stdout, "", fmt.Errorf("--user is required"))
			}
			token, err := clusterauth.ResolveToken(c, hooksKubectlAuthCluster, hooksKubectlAuthUser)
			return kubectlReturn(os.Stdout, token, err)
		},
	}
	cmd.Flags().StringVarP(&hooksKubectlAuthCluster, "cluster", "c", "", "Cluster ID (required)")
	cmd.Flags().StringVarP(&hooksKubectlAuthUser, "user", "u", "", "User sub (required)")
	return cmd
}
