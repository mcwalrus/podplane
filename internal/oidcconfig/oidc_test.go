// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package oidcconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidAWSConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "podplane.oidc.jsonc")
	if err := os.WriteFile(path, []byte(`{
  // comments are allowed
  "oidc": {
    "provider": { "kind": "aws", "region": "us-east-1", "account": "123456789012", "profile": "default" },
    "hostname": "auth.example.com",
    "domain": { "zone": "example.com", "provider": { "kind": "aws" } },
    "connector": {
      "kind": "google",
      "client_secret_arn": "arn:aws:secretsmanager:us-east-1:123456789012:secret:connector"
    },
    "signing_key_secret_arn": "arn:aws:secretsmanager:us-east-1:123456789012:secret:signing",
    "clients": {
      "kubelogin": { "groups_override": "prod" }
    },
    "groups_overrides": {
      "prod": { "ops@example.com": ["admins"] }
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.OIDC.Provider.Kind != "aws" {
		t.Fatalf("provider kind = %q, want aws", cfg.OIDC.Provider.Kind)
	}
}

func TestValidateRejectsGoogleProvider(t *testing.T) {
	err := Validate(&Config{OIDC: OIDC{
		Provider: Provider{Kind: "google", Region: "us-central1"},
	}})
	if err == nil {
		t.Fatal("Validate returned nil, want error")
	}
}
