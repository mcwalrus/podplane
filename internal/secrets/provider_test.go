// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"strings"
	"testing"

	"github.com/podplane/podplane/internal/clusterconfig"
	"github.com/podplane/podplane/internal/config"
)

// TestResolveProviderUsesRequestedProvider verifies explicit provider selection.
func TestResolveProviderUsesRequestedProvider(t *testing.T) {
	t.Parallel()
	summary := providerSummary()
	provider, err := ResolveProvider(summary, "openbao-local")
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if provider != "openbao-local" {
		t.Fatalf("ResolveProvider() = %q, want openbao-local", provider)
	}
}

// TestResolveProviderUsesDefaultProvider verifies default provider selection.
func TestResolveProviderUsesDefaultProvider(t *testing.T) {
	t.Parallel()
	summary := providerSummary()
	provider, err := ResolveProvider(summary, "")
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if provider != "aws-secrets-manager" {
		t.Fatalf("ResolveProvider() = %q, want aws-secrets-manager", provider)
	}
}

// TestResolveProviderRequiresDefaultProvider verifies missing default provider errors.
func TestResolveProviderRequiresDefaultProvider(t *testing.T) {
	t.Parallel()
	summary := providerSummary()
	summary.Secrets.DefaultProvider = ""
	_, err := ResolveProvider(summary, "")
	if err == nil || !strings.Contains(err.Error(), "no default secrets provider") {
		t.Fatalf("ResolveProvider() error = %v, want missing default error", err)
	}
}

// TestResolveProviderRequiresConfiguredProvider verifies unknown provider errors.
func TestResolveProviderRequiresConfiguredProvider(t *testing.T) {
	t.Parallel()
	_, err := ResolveProvider(providerSummary(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("ResolveProvider() error = %v, want not configured error", err)
	}
}

// providerSummary returns a cluster summary with two configured secrets providers.
func providerSummary() config.ClusterSummary {
	return config.ClusterSummary{
		ID: "cluster-a",
		Secrets: clusterconfig.Secrets{
			DefaultProvider: "aws-secrets-manager",
			Providers: map[string]clusterconfig.SecretsProvider{
				"aws-secrets-manager": {Kind: "aws"},
				"openbao-local":       {Kind: "openbao"},
			},
		},
	}
}
