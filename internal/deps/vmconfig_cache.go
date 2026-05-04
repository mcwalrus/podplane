// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// manifestFilename returns the manifest filename used both as part of the
// remote URL path and the local cache path, e.g. "knc.debian-13.arm64.json".
func manifestFilename(kind, arch string) string {
	return fmt.Sprintf("%s.%s.%s.json", kind, OS, arch)
}

// manifestName returns the manifest filename without its .json suffix.
func manifestName(kind, arch string) string {
	return strings.TrimSuffix(manifestFilename(kind, arch), ".json")
}

// VMConfigCacheDir returns the vmconfig dependency-domain cache directory.
func (m *Manager) VMConfigCacheDir() string {
	return filepath.Join(m.depsCacheDir, "vmconfig")
}

// VMConfigManifestCacheDir returns the vmconfig manifest cache directory.
func (m *Manager) VMConfigManifestCacheDir() string {
	return filepath.Join(m.VMConfigCacheDir(), "manifests")
}

// VMConfigArtifactsCacheDir returns the vmconfig artifact cache directory.
func (m *Manager) VMConfigArtifactsCacheDir() string {
	return filepath.Join(m.VMConfigCacheDir(), "artifacts")
}

// VMConfigManifestCachePath returns the path to the cached vmconfig manifest
// JSON file, e.g. ~/.podplane/cache/deps/vmconfig/manifests/knc.debian-13.arm64.json.
func (m *Manager) VMConfigManifestCachePath(kind, arch string) string {
	return filepath.Join(m.VMConfigManifestCacheDir(), manifestFilename(kind, arch))
}

// VMConfigHistoricalManifestCachePath returns an immutable historical manifest path.
func (m *Manager) VMConfigHistoricalManifestCachePath(kind, arch string, raw []byte) string {
	return filepath.Join(m.VMConfigManifestCacheDir(), historicalManifestFilename(manifestName(kind, arch), raw))
}

// VMConfigArtifactCachePath returns the cache path for a single named vmconfig
// dependency artifact, e.g.
// "<depsCacheDir>/vmconfig/artifacts/<name>/<version>/<basename(url)>". This is the
// single source of truth for where downloaded vmconfig artifacts live; both
// Download and Verify use it.
func (m *Manager) VMConfigArtifactCachePath(name string, dep Dependency) string {
	return filepath.Join(m.VMConfigArtifactsCacheDir(), name, dep.Version, path.Base(dep.URL))
}

// ReadCachedManifest reads and parses the cached manifest JSON. It returns
// (nil, nil, nil) if the cache file does not exist.
func (m *Manager) ReadCachedManifest(kind, arch string) (*Manifest, []byte, error) {
	path := m.VMConfigManifestCachePath(kind, arch)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read cached manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, raw, fmt.Errorf("failed to parse cached manifest: %w", err)
	}
	return &manifest, raw, nil
}

// WriteCachedManifest writes historical and latest manifest JSON bytes to the cache.
func (m *Manager) WriteCachedManifest(kind, arch string, raw []byte) error {
	historicalPath := m.VMConfigHistoricalManifestCachePath(kind, arch, raw)
	if err := os.MkdirAll(filepath.Dir(historicalPath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(historicalPath, raw, 0644); err != nil {
		return fmt.Errorf("failed to write historical cached manifest: %w", err)
	}
	latestPath := m.VMConfigManifestCachePath(kind, arch)
	if err := os.WriteFile(latestPath, raw, 0644); err != nil {
		return fmt.Errorf("failed to write cached manifest: %w", err)
	}
	return nil
}

// historicalManifestFilename returns a timestamped content-addressed manifest filename.
func historicalManifestFilename(name string, raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%s-%s-%s.json", name, time.Now().UTC().Format("20060102T150405Z"), hex.EncodeToString(sum[:])[:12])
}
