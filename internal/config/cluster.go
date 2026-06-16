// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/podplane/podplane/internal/clusterconfig"
)

// ClusterSummary is the subset of podplane.cluster.jsonc cached for commands
// that run from kube context instead of a cluster config file.
type ClusterSummary struct {
	Cluster ClusterSummaryCluster `mapstructure:"cluster" json:"cluster"`
}

// ClusterSummaryCluster is the subset of clusterconfig.Cluster persisted in the
// CLI config file.
type ClusterSummaryCluster struct {
	ID         string                          `mapstructure:"id" json:"id"`
	Name       string                          `mapstructure:"name" json:"name"`
	OIDC       clusterconfig.OIDC              `mapstructure:"oidc" json:"oidc"`
	Kubernetes clusterconfig.Kubernetes        `mapstructure:"kubernetes" json:"kubernetes"`
	Components ClusterSummaryClusterComponents `mapstructure:"components" json:"components,omitempty"`
}

// ClusterSummaryClusterComponents is the subset of clusterconfig.Components
// persisted in the CLI config file.
type ClusterSummaryClusterComponents struct {
	Registry *clusterconfig.ComponentsRegistry `mapstructure:"registry" json:"registry,omitempty"`
}

// ClusterSummaryFromConfig extracts the cached cluster summary from a full
// cluster config.
func ClusterSummaryFromConfig(cluster *clusterconfig.ClusterConfig) ClusterSummary {
	return ClusterSummary{
		Cluster: ClusterSummaryCluster{
			ID:         cluster.Cluster.ID,
			Name:       cluster.Cluster.Name,
			OIDC:       cluster.Cluster.OIDC,
			Kubernetes: cluster.Cluster.Kubernetes,
			Components: ClusterSummaryClusterComponents{
				Registry: cluster.Cluster.Components.Registry,
			},
		},
	}
}

// ClusterSummary returns the cached cluster summary for clusterID. Missing
// entries return a zero-value ClusterSummary and no error.
func (c *Config) ClusterSummary(clusterID string) (ClusterSummary, error) {
	var summary ClusterSummary
	if clusterID == "" {
		return summary, fmt.Errorf("ClusterSummary: cluster_id is required")
	}
	if raw := c.viperFile.GetStringMap("clusters." + clusterID); len(raw) > 0 {
		if err := decodeMap(raw, &summary); err != nil {
			return ClusterSummary{}, fmt.Errorf("decode cluster summary for %s: %w", clusterID, err)
		}
	}
	return summary, nil
}

// SetClusterSummary writes the cached cluster summary.
func (c *Config) SetClusterSummary(summary ClusterSummary) error {
	if summary.Cluster.ID == "" {
		return fmt.Errorf("SetClusterSummary: cluster.id is required")
	}
	cluster := map[string]any{
		"id":         summary.Cluster.ID,
		"name":       summary.Cluster.Name,
		"oidc":       summary.Cluster.OIDC,
		"kubernetes": summary.Cluster.Kubernetes,
	}
	if summary.Cluster.Components.Registry != nil {
		cluster["components"] = map[string]any{
			"registry": summary.Cluster.Components.Registry,
		}
	}
	c.viperFile.Set("clusters."+summary.Cluster.ID, map[string]any{"cluster": cluster})
	if err := c.SaveFile(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
