// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"reflect"
	"testing"

	"github.com/podplane/podplane/internal/clusterconfig"
)

func TestSetClusterSummary(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Init()
	if err != nil {
		t.Fatal(err)
	}

	summary := ClusterSummary{
		Cluster: ClusterSummaryCluster{
			ID:   "test-cluster",
			Name: "Test Cluster",
			OIDC: clusterconfig.OIDC{IssuerURL: "https://auth.example.com", ClientID: "test-client"},
			Kubernetes: clusterconfig.Kubernetes{
				APIHostname: "api.example.com",
				APIPort:     6443,
			},
			Components: ClusterSummaryClusterComponents{
				Registry: &clusterconfig.ComponentsRegistry{
					Mirror: clusterconfig.ComponentsRegistryMirror{Enabled: true, Hostname: "zot.local"},
				},
			},
		},
	}
	if err := c.SetClusterSummary(summary); err != nil {
		t.Fatal(err)
	}
	got, err := c.ClusterSummary("test-cluster")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, summary) {
		t.Fatalf("ClusterSummary() = %#v, want %#v", got, summary)
	}
}

func TestClusterSummaryMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Init()
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ClusterSummary("missing")
	if err != nil {
		t.Fatal(err)
	}
	if got.Cluster.ID != "" {
		t.Fatalf("ClusterSummary() = %#v, want missing summary", got)
	}
}
