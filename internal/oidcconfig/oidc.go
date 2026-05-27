// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package oidcconfig

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tailscale/hujson"
)

// Load reads an OIDC config file, strips JSONC comments, and validates the
// parsed configuration.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OIDC config %s: %w", path, err)
	}
	standardized, err := hujson.Standardize(raw)
	if err != nil {
		return nil, fmt.Errorf("parse OIDC config %s: %w", path, err)
	}
	cfg := &Config{}
	if err := json.Unmarshal(standardized, cfg); err != nil {
		return nil, fmt.Errorf("parse OIDC config %s: %w", path, err)
	}
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("OIDC config %s: %w", path, err)
	}
	return cfg, nil
}

// Write writes a formatted OIDC configuration file to disk.
func Write(path string, cfg *Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal OIDC config: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write OIDC config %s: %w", path, err)
	}
	return nil
}
