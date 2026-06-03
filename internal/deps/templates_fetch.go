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

// fetchTemplatesManifest downloads and parses the published templates manifest.
func (m *Manager) fetchTemplatesManifest(ctx context.Context, client *http.Client) (*TemplatesManifest, error) {
	url := fmt.Sprintf("%s/%s", m.baseURL, templatesManifestPath)
	var manifest TemplatesManifest
	if err := fetchDependencyManifestJSON(ctx, client, url, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// readTemplatesManifestFile reads and parses a local templates manifest file.
func readTemplatesManifestFile(path string) (*TemplatesManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest TemplatesManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
