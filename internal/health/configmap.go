// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/podplane/podplane/internal/execwrap"
	"github.com/podplane/podplane/internal/kubectl"
)

// ConfigMapDataCheck returns a health check for a required ConfigMap data key.
func ConfigMapDataCheck(kubeContext, kubeconfig, namespace, name, dataKey string, required bool) Check {
	return Check{
		Key:      key(namespace, "configmap", name),
		Name:     name,
		Kind:     "configmap",
		Required: required,
		Run: func(ctx context.Context) Result {
			return checkConfigMapData(ctx, kubeContext, kubeconfig, namespace, name, dataKey)
		},
	}
}

// checkConfigMapData reads one ConfigMap and verifies the requested data key is
// present and non-empty.
func checkConfigMapData(ctx context.Context, kubeContext, kubeconfig, namespace, name, dataKey string) Result {
	if err := ctx.Err(); err != nil {
		return Result{Err: err}
	}
	args := kubectl.Args(kubeContext, kubeconfig)
	args = append(args, "-n", namespace, "get", "configmap", name, "-o", "json")
	var stdout, stderr bytes.Buffer
	cmd := execwrap.Command("kubectl", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "NotFound") || strings.Contains(stderr.String(), "not found") {
			return Result{Status: StatusPending, Message: "waiting for configmap to be created"}
		}
		return Result{Err: &kubectl.Error{Stage: "get configmap", Err: err, Stderr: stderr.String()}}
	}

	var obj configMapObject
	if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
		return Result{Err: fmt.Errorf("decode configmap data: %w", err)}
	}
	if strings.TrimSpace(obj.Data[dataKey]) == "" {
		return Result{Exists: true, Status: StatusPending, Message: fmt.Sprintf("waiting for %s data", dataKey)}
	}
	return Result{Exists: true, Ready: true, Status: StatusReady, Message: fmt.Sprintf("%s data populated", dataKey)}
}

// configMapObject is the subset of Kubernetes ConfigMap JSON used by health
// checks.
type configMapObject struct {
	Data map[string]string `json:"data"`
}
