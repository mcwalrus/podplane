// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"path/filepath"
	"testing"
)

func TestCacheDomainPaths(t *testing.T) {
	manager := NewManager("https://example.invalid/deps", "/cache/deps")
	dep := Dependency{
		Version: "1.2.3",
		URL:     "https://example.invalid/artifacts/runc.arm64",
	}

	if got, want := manager.VMConfigCacheDir(), filepath.Join("/cache/deps", "vmconfig"); got != want {
		t.Fatalf("VMConfigCacheDir() = %q, want %q", got, want)
	}
	if got, want := manager.VMConfigManifestCacheDir(), filepath.Join("/cache/deps", "vmconfig", "manifests"); got != want {
		t.Fatalf("VMConfigManifestCacheDir() = %q, want %q", got, want)
	}
	if got, want := manager.VMConfigArtifactsCacheDir(), filepath.Join("/cache/deps", "vmconfig", "artifacts"); got != want {
		t.Fatalf("VMConfigArtifactsCacheDir() = %q, want %q", got, want)
	}
	if got, want := manager.VMConfigManifestCachePath(DefaultKind, "arm64"), filepath.Join("/cache/deps", "vmconfig", "manifests", "knc.debian-13.arm64.json"); got != want {
		t.Fatalf("VMConfigManifestCachePath() = %q, want %q", got, want)
	}
	if got, want := manager.VMConfigArtifactCachePath("runc", dep), filepath.Join("/cache/deps", "vmconfig", "artifacts", "runc", "1.2.3", "runc.arm64"); got != want {
		t.Fatalf("VMConfigArtifactCachePath() = %q, want %q", got, want)
	}
	if got, want := manager.ComponentsCacheDir(), filepath.Join("/cache/deps", "components"); got != want {
		t.Fatalf("ComponentsCacheDir() = %q, want %q", got, want)
	}
	if got, want := manager.ComponentsManifestCacheDir(), filepath.Join("/cache/deps", "components", "manifests"); got != want {
		t.Fatalf("ComponentsManifestCacheDir() = %q, want %q", got, want)
	}
	if got, want := manager.ComponentsImagesCacheDir(), filepath.Join("/cache/deps", "components", "images"); got != want {
		t.Fatalf("ComponentsImagesCacheDir() = %q, want %q", got, want)
	}
	if got, want := manager.ComponentsManifestCachePath(), filepath.Join("/cache/deps", "components", "manifests", "components.json"); got != want {
		t.Fatalf("ComponentsManifestCachePath() = %q, want %q", got, want)
	}
}

func TestManifestFilename(t *testing.T) {
	// The filename is part of the URL contract with the deps server, so a
	// regression here would silently break manifest fetching.
	tests := []struct {
		name string
		kind string
		arch string
		want string
	}{
		{
			name: "knc arm64",
			kind: "knc",
			arch: "arm64",
			want: "knc.debian-13.arm64.json",
		},
		{
			name: "knc amd64",
			kind: "knc",
			arch: "amd64",
			want: "knc.debian-13.amd64.json",
		},
		{
			name: "default kind constant",
			kind: DefaultKind,
			arch: "arm64",
			want: "knc.debian-13.arm64.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manifestFilename(tt.kind, tt.arch)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
