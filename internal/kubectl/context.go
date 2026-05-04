// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package kubectl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/podplane/podplane/internal/execwrap"
)

// SetContext first uses kubectl to check if a context with the same key already
// exists (and early exits if so), then runs kubectl to set a context
func SetContext(stdout io.Writer, sub string, clusterID string, local bool) error {
	// build kubectl keys
	key := ContextKey(clusterID, local)
	clusterKey := ClusterKey(clusterID, local)
	credentialsKey := CredentialsKey(sub, clusterID, local)
	// run kubectl to check if context already exists
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := execwrap.Command(
		"kubectl",
		"config",
		"view",
		"--output=jsonpath={.contexts[?(@.name==\""+key+"\")].context}",
	)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		stderr := strings.TrimSpace(errBuf.String())
		// Handle the specific JSONPath error when contexts array is nil (first-time setup)
		if strings.Contains(stderr, "is not array or slice and cannot be filtered") {
			// This is expected for first-time setup, continue with context configuration
		} else {
			return fmt.Errorf("error invoking kubectl command: %s", err)
		}
	} else {
		// parse json output
		outString := strings.TrimSpace(outBuf.String())
		var outJson struct {
			Cluster string `json:"cluster"`
			User    string `json:"user"`
		}
		err := json.Unmarshal([]byte(outString), &outJson)
		if err == nil {
			if outJson.Cluster != clusterKey {
				fmt.Fprintf(stdout, "Skipping configuration of kubectl context '%s' due to cluster mismatch.\n", key)
				return nil
			} else if outJson.User == credentialsKey {
				fmt.Fprintf(stdout, "Context already exist for %s\n", key)
				return nil
			}
		}
	}
	// run kubectl to configure context
	cmd = execwrap.Command(
		"kubectl",
		"config",
		"set-context",
		key,
		"--cluster="+clusterKey,
		"--user="+credentialsKey,
		"--namespace=default",
	)
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
