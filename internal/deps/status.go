// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"bytes"
	"context"
	"encoding/json"
)

// Status fetches the latest manifest and compares it byte-for-byte against
// the cached copy. It returns true if the cached copy is missing or differs
// from the latest, and also returns the parsed latest manifest.
func (m *Manager) Status(kind, arch string) (bool, *Manifest, error) {
	latest, err := m.fetchManifest(context.Background(), kind, arch, nil)
	if err != nil {
		return false, nil, err
	}

	cached, _, err := m.ReadCachedManifest(kind, arch)
	if err != nil {
		return false, latest, err
	}

	latestRaw, err := manifestComparableJSON(latest)
	if err != nil {
		return false, latest, err
	}
	cachedRaw, err := manifestComparableJSON(cached)
	if err != nil {
		return false, latest, err
	}
	needsUpdate := !bytes.Equal(bytes.TrimSpace(cachedRaw), bytes.TrimSpace(latestRaw))
	return needsUpdate, latest, nil
}

// manifestComparableJSON returns a stable JSON representation for comparing
// upstream manifest contents while ignoring local cache-state annotations.
func manifestComparableJSON(manifest *Manifest) ([]byte, error) {
	if manifest == nil {
		return nil, nil
	}
	copy := *manifest
	copy.VMConfig.OS.Image.Cached = false
	copy.VMConfig.Dependencies = make(map[string]Dependency, len(manifest.VMConfig.Dependencies))
	for name, dep := range manifest.VMConfig.Dependencies {
		dep.Cached = false
		copy.VMConfig.Dependencies[name] = dep
	}
	return json.Marshal(copy)
}
