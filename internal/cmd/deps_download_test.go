// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"reflect"
	"testing"
)

// TestInferredSecretsProvidersFromInfraProviders verifies infra providers imply secrets provider downloads.
func TestInferredSecretsProvidersFromInfraProviders(t *testing.T) {
	got := inferredSecretsProviders([]string{"aws", "gcp,proxmox"})
	want := []string{"aws", "gcp", "openbao"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inferredSecretsProviders() = %v, want %v", got, want)
	}
}
