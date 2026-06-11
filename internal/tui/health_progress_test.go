// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"testing"
	"time"

	"github.com/podplane/podplane/internal/health"
)

// TestHealthCriticalPathExpected uses the longest dependency path rather than
// summing every health check expectation.
func TestHealthCriticalPathExpected(t *testing.T) {
	checks := []health.Check{
		{Key: "cilium", Required: true, Expected: 30 * time.Second},
		{Key: "cert-manager", Required: true, DependsOn: []string{"cilium"}, Expected: 30 * time.Second},
		{Key: "cert-manager-admission", Required: true, DependsOn: []string{"cert-manager"}, Expected: 10 * time.Second},
		{Key: "traefik", Required: true, DependsOn: []string{"cilium"}, Expected: 20 * time.Second},
		{Key: "ingress", Required: true, DependsOn: []string{"traefik"}, Expected: 5 * time.Second},
	}

	if got, want := healthCriticalPathExpected(checks), 70*time.Second; got != want {
		t.Fatalf("healthCriticalPathExpected() = %s, want %s", got, want)
	}
}
