// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/podplane/podplane/internal/config"
)

const validClusterConfigJSON = `{
  "cluster": {
    "id": "test-cluster",
    "oidc": { "issuer_url": "https://auth.example.com" },
    "pools": {
      "control-plane": { "arch": "arm64", "instance_type": "t4g.medium", "size": 1 }
    },
    "providers": [{
      "kind": "aws",
      "region": "us-east-1",
      "account": "123456789012",
      "vpc": { "v4cidr": "172.18.0.0/16", "v6cidr": "auto" },
      "zones": {
        "us-east-1a": [
          { "v4cidr": "172.18.10.0/28", "services": ["nat", "nlb"], "public": true },
          { "v4cidr": "172.18.20.0/28", "services": ["nstance"] },
          { "v4cidr": "172.18.1.0/24", "pool": "control-plane" }
        ]
      },
      "load_balancer": {
        "public": true,
        "listeners": [{ "port": 6443, "pool": "control-plane" }]
      }
    }],
    "kubernetes": {
      "cluster_cidr": ["100.64.0.0/10"],
      "service_cidr": ["198.18.0.0/15"]
    }
  }
}`

const validOIDCConfigJSON = `{
  "oidc": {
    "provider": { "kind": "aws", "region": "us-east-1", "account": "123456789012" },
    "hostname": "auth.example.com",
    "domain": { "zone": "example.com", "provider": { "kind": "aws" } },
    "connector": { "kind": "google", "client_secret_arn": "arn:connector" },
    "signing_key_secret_arn": "arn:signing",
    "clients": { "kubelogin": {} }
  }
}`

// TestClusterCreateNoApplyGeneratesTerraform verifies cluster create writes
// managed Terraform without invoking OpenTofu/Terraform.
func TestClusterCreateNoApplyGeneratesTerraform(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podplane.cluster.jsonc")
	if err := os.WriteFile(path, []byte(validClusterConfigJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newClusterCreateCmd(&config.Config{})
	cmd.SetArgs([]string{"--cluster-config", path, "--no-apply"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cluster create --no-apply returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "podplane.cluster.tf")); err != nil {
		t.Fatalf("AWS cluster tf was not generated: %v", err)
	}
}

// TestOIDCCreateNoApplyGeneratesTerraform verifies OIDC create writes managed
// Terraform without invoking OpenTofu/Terraform.
func TestOIDCCreateNoApplyGeneratesTerraform(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podplane.oidc.jsonc")
	if err := os.WriteFile(path, []byte(validOIDCConfigJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newOIDCCreateCmd(&config.Config{})
	cmd.SetArgs([]string{"--oidc-config", path, "--no-apply"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("oidc create --no-apply returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "podplane.oidc.tf")); err != nil {
		t.Fatalf("OIDC tf was not generated: %v", err)
	}
}

// TestClusterDeleteNoApplyValidatesOnly verifies cluster delete no-apply
// validates the config without invoking destroy dependencies.
func TestClusterDeleteNoApplyValidatesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podplane.cluster.jsonc")
	if err := os.WriteFile(path, []byte(validClusterConfigJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newClusterDeleteCmd(&config.Config{})
	cmd.SetArgs([]string{"--cluster-config", path, "--no-apply"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cluster delete --no-apply returned error: %v", err)
	}
}

// TestOIDCDeleteNoApplyValidatesOnly verifies OIDC delete no-apply validates
// the config without invoking destroy dependencies.
func TestOIDCDeleteNoApplyValidatesOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podplane.oidc.jsonc")
	if err := os.WriteFile(path, []byte(validOIDCConfigJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newOIDCDeleteCmd(&config.Config{})
	cmd.SetArgs([]string{"--oidc-config", path, "--no-apply"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("oidc delete --no-apply returned error: %v", err)
	}
}
