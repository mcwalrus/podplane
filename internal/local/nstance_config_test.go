// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"testing"
)

func TestConfigureLocalNstancePreparesBootstrap(t *testing.T) {
	bootstrap, err := configureLocalNstance(
		context.Background(),
		t.TempDir(),
		"cluster-a",
		"knc123",
		"knc",
		"10.0.2.2:1234",
		"10.0.2.2:5678",
		"10.0.2.2",
	)
	if err != nil {
		t.Fatalf("configureLocalNstance: %v", err)
	}
	if bootstrap.CACert == "" {
		t.Fatal("expected CA cert")
	}
	if bootstrap.RegistrationNonceJWT == "" {
		t.Fatal("expected registration nonce JWT")
	}
	if bootstrap.ServerRegistrationAddr != "10.0.2.2:1234" {
		t.Fatalf("registration addr = %q", bootstrap.ServerRegistrationAddr)
	}
	if bootstrap.ServerAgentAddr != "10.0.2.2:5678" {
		t.Fatalf("agent addr = %q", bootstrap.ServerAgentAddr)
	}
}

func TestPodplaneRuntimeConfigIncludesInternalKubeAPISAN(t *testing.T) {
	cfg := podplaneRuntimeConfig("cluster-a", nil)
	cert := cfg.Certificates["kube-apiserver.server"]
	for _, name := range cert.DNS {
		if name == "kube-apiserver.podplane.internal" {
			return
		}
	}
	t.Fatalf("kube-apiserver.server DNS SANs = %v, want kube-apiserver.podplane.internal", cert.DNS)
}
