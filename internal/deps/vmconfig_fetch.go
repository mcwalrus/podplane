// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// fetchManifest downloads and parses the latest vmconfig manifest from
// <baseURL>/vmconfig/manifests/<kind>.<os>.<arch>.json using client.
func (m *Manager) fetchManifest(ctx context.Context, kind, arch string, client *http.Client) (*Manifest, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	url := fmt.Sprintf("%s/vmconfig/manifests/%s", m.baseURL, manifestFilename(kind, arch))
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest from %s: HTTP status %d", url, resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest body: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON from %s: %w", url, err)
	}

	return &manifest, nil
}

func readVMConfigManifestFile(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read vmconfig manifest file %s: %w", path, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse vmconfig manifest JSON from %s: %w", path, err)
	}

	return &manifest, nil
}
