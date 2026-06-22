// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"fmt"
	"strings"

	"github.com/podplane/podplane/internal/config"
)

// ResolveProvider returns the requested secrets provider or the cluster default.
func ResolveProvider(summary config.ClusterSummary, requested string) (string, error) {
	provider := strings.TrimSpace(requested)
	if provider == "" {
		provider = summary.Secrets.DefaultProvider
		if provider == "" {
			return "", fmt.Errorf("no default secrets provider is cached for cluster %q; pass --provider or rerun login/local start after configuring cluster.secrets", summary.ID)
		}
	}
	if err := ValidateProviderName(provider); err != nil {
		return "", err
	}
	if !SummaryHasProvider(summary, provider) {
		return "", fmt.Errorf("secrets provider %q is not configured for cluster %q", provider, summary.ID)
	}
	return provider, nil
}

// SummaryHasProvider reports whether the cached cluster summary includes provider.
func SummaryHasProvider(summary config.ClusterSummary, provider string) bool {
	_, ok := summary.Secrets.Providers[provider]
	return ok
}
