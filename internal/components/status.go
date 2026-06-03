// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/podplane/podplane/internal/execwrap"
)

// InstallItem identifies one component HelmRelease that is expected to exist
// after enabling a component install plan.
type InstallItem struct {
	Name      string
	Namespace string
	Kind      InstallItemKind
}

// InstallItemKind identifies whether an install status item is an app chart or
// a CRD chart managed by platform-components.
type InstallItemKind string

const (
	// InstallItemApp is a component app HelmRelease.
	InstallItemApp InstallItemKind = "app"
	// InstallItemCRD is a component CRD HelmRelease.
	InstallItemCRD InstallItemKind = "crd"
)

// InstallStatus is the current observed status of one component HelmRelease.
type InstallStatus struct {
	Item    InstallItem
	Exists  bool
	Ready   bool
	Status  string
	Message string
}

// InstallItems builds watch targets for every app and CRD in an enable plan.
func (c *Config) InstallItems(plan EnableSet) []InstallItem {
	items := make([]InstallItem, 0, len(plan.CRDs)+len(plan.Apps))
	for _, name := range plan.CRDs {
		items = append(items, InstallItem{Name: name, Namespace: "platform-cluster", Kind: InstallItemCRD})
	}
	for _, name := range plan.Apps {
		entry, _, _ := c.Get(name)
		ns := entry.Namespace
		if ns == "" {
			ns = HelmReleaseNamespace
		}
		items = append(items, InstallItem{Name: name, Namespace: ns, Kind: InstallItemApp})
	}
	return items
}

// ReadInstallStatuses reads all Flux HelmRelease statuses once and returns the
// current status for the requested items. kubeContext and kubeconfig are
// optional; empty values use kubectl's defaults.
func ReadInstallStatuses(ctx context.Context, kubeContext, kubeconfig string, items []InstallItem) (map[string]InstallStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	statuses := map[string]InstallStatus{}
	for _, item := range items {
		statuses[itemKey(item)] = InstallStatus{
			Item:    item,
			Status:  "Pending",
			Message: "waiting for HelmRelease to be created",
		}
	}
	if len(items) == 0 {
		return statuses, nil
	}

	args := []string{"get", "helmreleases.helm.toolkit.fluxcd.io", "-A", "-o", "json"}
	args = append(kubectlArgs(kubeContext, kubeconfig), args...)
	var stdout, stderr bytes.Buffer
	cmd := execwrap.Command("kubectl", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, &KubectlError{Stage: "get helmreleases", Err: err, Stderr: stderr.String()}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var list helmReleaseList
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return nil, fmt.Errorf("decode HelmRelease status: %w", err)
	}
	for _, hr := range list.Items {
		key := hr.Metadata.Namespace + "/" + hr.Metadata.Name
		status, ok := statuses[key]
		if !ok {
			continue
		}
		status.Exists = true
		status.Ready, status.Status, status.Message = helmReleaseReadyStatus(hr.Status.Conditions)
		statuses[key] = status
	}
	return statuses, nil
}

// RequiredItemsReady reports whether every named required item is Ready in the
// supplied status snapshot.
func RequiredItemsReady(statuses map[string]InstallStatus, required []InstallItem) bool {
	for _, item := range required {
		if !statuses[itemKey(item)].Ready {
			return false
		}
	}
	return true
}

// itemKey returns the namespace/name key used to index install status maps.
func itemKey(item InstallItem) string {
	return item.Namespace + "/" + item.Name
}

// helmReleaseReadyStatus converts Flux HelmRelease Ready conditions into the
// compact readiness tuple displayed by the dependency install UI.
func helmReleaseReadyStatus(conditions []helmReleaseCondition) (bool, string, string) {
	var ready *helmReleaseCondition
	for i := range conditions {
		if conditions[i].Type == "Ready" {
			ready = &conditions[i]
			break
		}
	}
	if ready == nil {
		return false, "Reconciling", "waiting for Ready condition"
	}
	message := strings.TrimSpace(ready.Message)
	if ready.Status == "True" {
		if message == "" {
			message = "HelmRelease is ready"
		}
		return true, "Ready", message
	}
	status := ready.Reason
	if status == "" {
		status = "Reconciling"
	}
	if message == "" {
		message = "waiting for HelmRelease to become ready"
	}
	return false, status, message
}

type helmReleaseList struct {
	Items []helmRelease `json:"items"`
}

type helmRelease struct {
	Metadata helmReleaseMetadata `json:"metadata"`
	Status   helmReleaseStatus   `json:"status"`
}

type helmReleaseMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type helmReleaseStatus struct {
	Conditions []helmReleaseCondition `json:"conditions"`
}

type helmReleaseCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}
