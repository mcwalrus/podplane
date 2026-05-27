// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseHelmRelease(t *testing.T) {
	obj := map[string]any{
		"apiVersion": "helm.toolkit.fluxcd.io/v2",
		"kind":       "HelmRelease",
		"metadata": map[string]any{
			"name":      "platform-components",
			"namespace": "platform-components",
		},
		"spec": map[string]any{
			"values": map[string]any{
				"platform": map[string]any{
					"components": map[string]any{
						"crds": map[string]any{
							"cilium-crds":       map[string]any{"enabled": true},
							"cert-manager-crds": map[string]any{"enabled": false},
						},
						"apps": map[string]any{
							"cilium": map[string]any{
								"enabled":   true,
								"namespace": "platform-cilium",
								"dependsOn": []any{"cilium-crds"},
							},
							"cert-manager": map[string]any{
								"enabled":   false,
								"namespace": "platform-cert-manager",
								"dependsOn": []any{"cert-manager-crds"},
								// Explicit core value beats the fallback.
								"core": false,
							},
							"platform-rbac": map[string]any{
								"enabled":   true,
								"core":      true,
								"namespace": "platform-cluster",
							},
						},
					},
				},
			},
		},
	}
	// helm get values --all -o json returns the merged values structure
	// (no spec.values wrapper).
	values := obj["spec"].(map[string]any)["values"].(map[string]any)
	raw, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := parseValues(raw)
	if err != nil {
		t.Fatalf("parseValues: %v", err)
	}

	cilium, isApp, ok := cfg.Get("cilium")
	if !ok || !isApp {
		t.Fatalf("expected cilium app entry, got isApp=%v ok=%v", isApp, ok)
	}
	if !cilium.Enabled {
		t.Errorf("cilium should be enabled")
	}
	if !reflect.DeepEqual(cilium.DependsOn, []string{"cilium-crds"}) {
		t.Errorf("cilium.DependsOn = %v", cilium.DependsOn)
	}

	cm, _, _ := cfg.Get("cert-manager")
	if cm.Core {
		t.Errorf("cert-manager.core = true, want false (explicit)")
	}

	rbac, _, _ := cfg.Get("platform-rbac")
	if !rbac.Core {
		t.Errorf("platform-rbac.core = false, want true (explicit)")
	}

	if _, isApp, ok := cfg.Get("cilium-crds"); !ok || isApp {
		t.Fatalf("expected cilium-crds crd entry, got isApp=%v ok=%v", isApp, ok)
	}
}

func TestParseValuesMissingComponents(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"platform": map[string]any{}})
	if _, err := parseValues(raw); err == nil {
		t.Fatal("expected error for missing platform.components")
	}
}
