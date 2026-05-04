// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package clusterconfig

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tidwall/jsonc"
)

// Load reads a .cluster.jsonc file from disk, strips comments, and unmarshals
// it into a ClusterConfig.
func Load(path string) (*ClusterConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cluster config %s: %w", path, err)
	}
	cfg := &ClusterConfig{}
	if err := json.Unmarshal(jsonc.ToJSON(raw), cfg); err != nil {
		return nil, fmt.Errorf("parse cluster config %s: %w", path, err)
	}
	if cfg.Cluster.ID == "" {
		return nil, fmt.Errorf("cluster config %s: missing cluster.id", path)
	}
	if err := ValidateClusterID(cfg.Cluster.ID); err != nil {
		return nil, fmt.Errorf("cluster config %s: invalid cluster.id: %w", path, err)
	}
	if cfg.Cluster.OIDC.IssuerURL == "" {
		return nil, fmt.Errorf("cluster config %s: missing cluster.oidc.issuer_url", path)
	}
	if err := ValidateComponents(cfg.Cluster.Components); err != nil {
		return nil, fmt.Errorf("cluster config %s: invalid cluster.components: %w", path, err)
	}
	return cfg, nil
}
