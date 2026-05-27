// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package components

import "testing"

func TestInstallItems(t *testing.T) {
	cfg := &Config{
		Apps: map[string]Entry{
			"cert-manager": {Namespace: "platform-cert-manager"},
		},
		CRDs: map[string]Entry{
			"cert-manager-crds": {},
		},
	}
	items := cfg.InstallItems(EnableSet{Apps: []string{"cert-manager"}, CRDs: []string{"cert-manager-crds"}})
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if got, want := items[0], (InstallItem{Name: "cert-manager-crds", Namespace: "platform-cluster", Kind: InstallItemCRD}); got != want {
		t.Fatalf("items[0] = %#v, want %#v", got, want)
	}
	if got, want := items[1], (InstallItem{Name: "cert-manager", Namespace: "platform-cert-manager", Kind: InstallItemApp}); got != want {
		t.Fatalf("items[1] = %#v, want %#v", got, want)
	}
}

func TestHelmReleaseReadyStatus(t *testing.T) {
	ready, status, message := helmReleaseReadyStatus([]helmReleaseCondition{{Type: "Ready", Status: "True"}})
	if !ready || status != "Ready" || message == "" {
		t.Fatalf("ready status = (%v, %q, %q), want ready with message", ready, status, message)
	}

	ready, status, message = helmReleaseReadyStatus([]helmReleaseCondition{{Type: "Ready", Status: "False", Reason: "Progressing", Message: "installing"}})
	if ready || status != "Progressing" || message != "installing" {
		t.Fatalf("not-ready status = (%v, %q, %q), want false Progressing installing", ready, status, message)
	}

	ready, status, message = helmReleaseReadyStatus(nil)
	if ready || status != "Reconciling" || message == "" {
		t.Fatalf("missing status = (%v, %q, %q), want false Reconciling with message", ready, status, message)
	}
}
