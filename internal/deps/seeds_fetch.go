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

const seedsManifestPath = "manifests/seeds.json"

// fetchSeedsManifest downloads and parses the published seeds manifest.
func (m *Manager) fetchSeedsManifest(ctx context.Context, client *http.Client) (*SeedsManifest, error) {
	url := fmt.Sprintf("%s/%s", m.baseURL, seedsManifestPath)
	var manifest SeedsManifest
	if err := fetchDependencyManifestJSON(ctx, client, url, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// readSeedsManifestFile reads and parses a local seeds manifest file.
func readSeedsManifestFile(path string) (*SeedsManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest SeedsManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
