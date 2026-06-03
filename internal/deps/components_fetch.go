// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const componentsManifestPath = "manifests/components.json"

// fetchComponentsManifest downloads and parses the published components manifest.
func (m *Manager) fetchComponentsManifest(ctx context.Context, client *http.Client) (*ComponentsManifest, error) {
	url := fmt.Sprintf("%s/%s", m.baseURL, componentsManifestPath)
	var manifest ComponentsManifest
	if err := fetchDependencyManifestJSON(ctx, client, url, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// readComponentsManifestFile reads and parses a local components manifest file.
func readComponentsManifestFile(path string) (*ComponentsManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest ComponentsManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
