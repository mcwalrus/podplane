// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package kubectl

import "testing"

func TestClusterKey(t *testing.T) {
	tests := []struct {
		clusterID string
		local     bool
		want      string
	}{
		{"my-cluster", false, "podplane-my-cluster"},
		{"default", false, "podplane-default"},
		{"my-cluster", true, "podplane-local-my-cluster"},
		{"default", true, "podplane-local-default"},
	}
	for _, tt := range tests {
		got := ClusterKey(tt.clusterID, tt.local)
		if got != tt.want {
			t.Errorf("ClusterKey(%q, %v) = %q, want %q", tt.clusterID, tt.local, got, tt.want)
		}
	}
}

func TestContextKey(t *testing.T) {
	tests := []struct {
		clusterID string
		local     bool
		want      string
	}{
		{"my-cluster", false, "my-cluster"},
		{"default", false, "default"},
		{"my-cluster", true, "local-my-cluster"},
		{"default", true, "local"},
	}
	for _, tt := range tests {
		got := ContextKey(tt.clusterID, tt.local)
		if got != tt.want {
			t.Errorf("ContextKey(%q, %v) = %q, want %q", tt.clusterID, tt.local, got, tt.want)
		}
	}
}

func TestCredentialsKey(t *testing.T) {
	tests := []struct {
		sub       string
		clusterID string
		local     bool
		want      string
	}{
		{"user1", "my-cluster", false, "podplane-my-cluster-user1"},
		{"user1", "default", false, "podplane-default-user1"},
		{"user1", "my-cluster", true, "podplane-local-my-cluster-user1"},
		{"user1", "default", true, "podplane-local-default-user1"},
	}
	for _, tt := range tests {
		got := CredentialsKey(tt.sub, tt.clusterID, tt.local)
		if got != tt.want {
			t.Errorf("CredentialsKey(%q, %q, %v) = %q, want %q", tt.sub, tt.clusterID, tt.local, got, tt.want)
		}
	}
}
