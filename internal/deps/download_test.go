// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadDownloadsArtifactsInParallel(t *testing.T) {
	imageBody := []byte("image artifact")
	depBody := []byte("dependency artifact")
	bothStarted := make(chan struct{})
	var active int32
	var released int32
	var componentsManifestFetched atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/components/manifests/components.json":
			componentsManifestFetched.Store(true)
			manifest := ComponentsManifest{
				Components: Components{
					Version: "test",
					Images:  []ComponentImage{},
				},
			}
			if err := json.NewEncoder(w).Encode(manifest); err != nil {
				t.Errorf("encode components manifest: %v", err)
			}
		case "/vmconfig/manifests/knc.debian-13.arm64.json":
			manifest := Manifest{VMConfig: VMConfig{
				Version: "test",
				Kind:    DefaultKind,
				OS: OSInfo{
					Name: OS,
					Arch: "arm64",
					Image: Dependency{
						Version: "1",
						URL:     serverURL(r, "/image"),
						Type:    "binary",
						Digest:  "sha256:" + sha256Hex(imageBody),
						Size:    int64(len(imageBody)),
					},
				},
				Dependencies: map[string]Dependency{
					"dep": {
						Version: "1",
						URL:     serverURL(r, "/dep"),
						Type:    "binary",
						Digest:  "sha256:" + sha256Hex(depBody),
						Size:    int64(len(depBody)),
					},
				},
			}}
			if err := json.NewEncoder(w).Encode(manifest); err != nil {
				t.Errorf("encode manifest: %v", err)
			}
		case "/image", "/dep":
			if !componentsManifestFetched.Load() {
				t.Errorf("artifact download started before components manifest was fetched")
				http.Error(w, "components manifest was not fetched first", http.StatusInternalServerError)
				return
			}
			current := atomic.AddInt32(&active, 1)
			if current == 2 && atomic.CompareAndSwapInt32(&released, 0, 1) {
				close(bothStarted)
			}
			select {
			case <-bothStarted:
			case <-time.After(2 * time.Second):
				t.Errorf("downloads did not run concurrently")
				http.Error(w, "timed out waiting for concurrent request", http.StatusInternalServerError)
				return
			}
			if r.URL.Path == "/image" {
				_, _ = w.Write(imageBody)
				return
			}
			_, _ = w.Write(depBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	manager := NewManager(server.URL, cacheDir)
	var doneEvents int32
	err := manager.Download(DefaultKind, "arm64", DownloadOptions{
		Concurrency: 2,
		Client:      server.Client(),
		Progress: func(event DownloadEvent) {
			if event.Type == DownloadEventDone {
				atomic.AddInt32(&doneEvents, 1)
			}
		},
	})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}

	if got := atomic.LoadInt32(&doneEvents); got != 2 {
		t.Fatalf("done events: got %d, want 2", got)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "vmconfig", "artifacts", "os-image", "1", "image")); err != nil {
		t.Fatalf("image was not cached: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "vmconfig", "artifacts", "dep", "1", "dep")); err != nil {
		t.Fatalf("dep was not cached: %v", err)
	}
	if _, err := os.Stat(manager.VMConfigManifestCachePath(DefaultKind, "arm64")); err != nil {
		t.Fatalf("manifest was not cached: %v", err)
	}
	if _, err := os.Stat(manager.ComponentsManifestCachePath()); err != nil {
		t.Fatalf("components manifest was not cached: %v", err)
	}
}

func TestDownloadUsesLocalManifestFile(t *testing.T) {
	imageBody := []byte("image artifact")
	depBody := []byte("dependency artifact")
	var manifestRequested atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/components/manifests/components.json":
			manifest := ComponentsManifest{
				Components: Components{
					Version: "test",
					Images:  []ComponentImage{},
				},
			}
			if err := json.NewEncoder(w).Encode(manifest); err != nil {
				t.Errorf("encode components manifest: %v", err)
			}
		case "/vmconfig/manifests/knc.debian-13.arm64.json":
			manifestRequested.Store(true)
			http.Error(w, "remote manifest should not be requested", http.StatusInternalServerError)
		case "/image":
			_, _ = w.Write(imageBody)
		case "/dep":
			_, _ = w.Write(depBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	manifest := Manifest{VMConfig: VMConfig{
		Version: "test",
		Kind:    DefaultKind,
		OS: OSInfo{
			Name: OS,
			Arch: "arm64",
			Image: Dependency{
				Version: "1",
				URL:     server.URL + "/image",
				Type:    "binary",
				Digest:  "sha256:" + sha256Hex(imageBody),
				Size:    int64(len(imageBody)),
			},
		},
		Dependencies: map[string]Dependency{
			"dep": {
				Version: "1",
				URL:     server.URL + "/dep",
				Type:    "binary",
				Digest:  "sha256:" + sha256Hex(depBody),
				Size:    int64(len(depBody)),
			},
			VMConfigDepName: {
				Version: "dev",
				URL:     "",
				Type:    "tar.gz",
				Digest:  "",
			},
		},
	}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	manifestPath := filepath.Join(t.TempDir(), "local-manifest.json")
	if err := os.WriteFile(manifestPath, raw, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cacheDir := t.TempDir()
	manager := NewManager(server.URL, cacheDir)
	if err := manager.Download(DefaultKind, "arm64", DownloadOptions{VMConfigManifestPath: manifestPath, Client: server.Client()}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if manifestRequested.Load() {
		t.Fatal("remote manifest was requested despite VMConfigManifestPath override")
	}
	if got, err := os.ReadFile(manager.VMConfigManifestCachePath(DefaultKind, "arm64")); err != nil {
		t.Fatalf("cached manifest was not written: %v", err)
	} else {
		var cached Manifest
		if err := json.Unmarshal(got, &cached); err != nil {
			t.Fatalf("parse cached manifest: %v", err)
		}
		if !cached.VMConfig.OS.Image.Cached {
			t.Fatal("cached manifest did not mark OS image cached")
		}
		if !cached.VMConfig.Dependencies["dep"].Cached {
			t.Fatal("cached manifest did not mark dep cached")
		}
		if _, ok := cached.VMConfig.Dependencies[VMConfigDepName]; !ok {
			t.Fatal("cached manifest should preserve vmconfig stub")
		}
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "vmconfig", "artifacts", "os-image", "1", "image")); err != nil {
		t.Fatalf("image was not cached: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "vmconfig", "artifacts", "dep", "1", "dep")); err != nil {
		t.Fatalf("dep was not cached: %v", err)
	}
	if _, err := manager.Verify(DefaultKind, "arm64"); err != nil {
		t.Fatalf("Verify returned error after downloading local manifest with vmconfig stub: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "vmconfig", "artifacts", VMConfigDepName, "dev")); !os.IsNotExist(err) {
		t.Fatalf("vmconfig stub should not be cached, stat err = %v", err)
	}
}

func TestDownloadRejectsPartiallyPopulatedVMConfigDependency(t *testing.T) {
	imageBody := []byte("image artifact")
	manifest := Manifest{VMConfig: VMConfig{
		Version: "test",
		Kind:    DefaultKind,
		OS: OSInfo{
			Name: OS,
			Arch: "arm64",
			Image: Dependency{
				Version: "1",
				URL:     "https://example.invalid/image",
				Type:    "binary",
				Digest:  "sha256:" + sha256Hex(imageBody),
				Size:    int64(len(imageBody)),
			},
		},
		Dependencies: map[string]Dependency{
			VMConfigDepName: {
				Version: "dev",
				URL:     "https://example.invalid/vmconfig.tar.gz",
				Type:    "tar.gz",
				Digest:  "",
			},
		},
	}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	manifestPath := filepath.Join(t.TempDir(), "local-manifest.json")
	if err := os.WriteFile(manifestPath, raw, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manager := NewManager("https://example.invalid", t.TempDir())
	err = manager.Download(DefaultKind, "arm64", DownloadOptions{DryRun: true, VMConfigManifestPath: manifestPath})
	if err == nil {
		t.Fatal("Download returned nil error, want invalid digest error")
	}
	if !strings.Contains(err.Error(), "invalid digest for vmconfig: missing digest") {
		t.Fatalf("Download error = %q, want vmconfig missing digest", err.Error())
	}
}

func TestDownloadRejectsDependencyMissingSize(t *testing.T) {
	body := []byte("image artifact")
	manifest := Manifest{VMConfig: VMConfig{
		Version: "test",
		Kind:    DefaultKind,
		OS: OSInfo{
			Name: OS,
			Arch: "arm64",
			Image: Dependency{
				Version: "1",
				URL:     "https://example.invalid/image",
				Type:    "binary",
				Digest:  "sha256:" + sha256Hex(body),
			},
		},
		Dependencies: map[string]Dependency{},
	}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	manifestPath := filepath.Join(t.TempDir(), "local-manifest.json")
	if err := os.WriteFile(manifestPath, raw, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manager := NewManager("https://example.invalid", t.TempDir())
	err = manager.Download(DefaultKind, "arm64", DownloadOptions{DryRun: true, VMConfigManifestPath: manifestPath})
	if err == nil {
		t.Fatal("Download returned nil error, want missing size error")
	}
	if !strings.Contains(err.Error(), "missing size for os-image") {
		t.Fatalf("Download error = %q, want missing size", err.Error())
	}
}

func TestFilterComponentsManifestExcludesAddonsAndProviderSpecificImagesByDefault(t *testing.T) {
	manifest := &ComponentsManifest{Components: Components{Images: []ComponentImage{
		{Component: "cilium", Image: "cilium"},
		{Component: "traefik", Image: "traefik", Addon: true},
		{Component: "csi-aws-ebs", Image: "aws-ebs", Providers: []string{"aws"}},
		{Component: "aws-addon", Image: "aws-addon", Addon: true, Providers: []string{"aws"}},
		{Component: "arm64", Image: "arm64", Platform: "linux/arm64/v8"},
	}}}
	archs := []string{"amd64"}

	indexes := manifest.DownloadImageIndexes(ComponentImageFilter{Archs: archs})
	if got := componentImageNamesAt(manifest.Components.Images, indexes); strings.Join(got, ",") != "cilium" {
		t.Fatalf("filtered images = %v, want cilium", got)
	}

	indexes = manifest.DownloadImageIndexes(ComponentImageFilter{Archs: archs, Providers: []string{"aws"}, Addons: []string{"traefik"}})
	if got := componentImageNamesAt(manifest.Components.Images, indexes); strings.Join(got, ",") != "cilium,traefik,aws-ebs" {
		t.Fatalf("filtered images = %v, want cilium,traefik,aws-ebs", got)
	}

	manifest.Components.Images = []ComponentImage{{Component: "aws-addon", Image: "aws-addon", Addon: true, Providers: []string{"aws"}}}
	indexes = manifest.DownloadImageIndexes(ComponentImageFilter{Archs: archs, Providers: []string{"all"}, Addons: []string{"all"}})
	if got := componentImageNamesAt(manifest.Components.Images, indexes); strings.Join(got, ",") != "aws-addon" {
		t.Fatalf("filtered images = %v, want aws-addon", got)
	}
}

func componentImageNamesAt(images []ComponentImage, indexes []int) []string {
	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		names = append(names, images[index].Image)
	}
	return names
}

func TestNormalizeArchsParsesCommaSeparatedValues(t *testing.T) {
	got, err := normalizeArchs([]string{"amd64, arm64", "amd64"}, "arm64")
	if err != nil {
		t.Fatalf("normalizeArchs returned error: %v", err)
	}
	if strings.Join(got, ",") != "amd64,arm64" {
		t.Fatalf("normalizeArchs = %v, want amd64,arm64", got)
	}

	if _, err := normalizeArchs([]string{"s390x"}, "arm64"); err == nil {
		t.Fatal("normalizeArchs returned nil error for unsupported arch")
	}
}

func TestDownloadRejectsRemoteManifestWithUnreleasedVMConfigStub(t *testing.T) {
	imageBody := []byte("image artifact")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vmconfig/manifests/knc.debian-13.arm64.json":
			manifest := Manifest{VMConfig: VMConfig{
				Version: "test",
				Kind:    DefaultKind,
				OS: OSInfo{
					Name: OS,
					Arch: "arm64",
					Image: Dependency{
						Version: "1",
						URL:     serverURL(r, "/image"),
						Type:    "binary",
						Digest:  "sha256:" + sha256Hex(imageBody),
						Size:    int64(len(imageBody)),
					},
				},
				Dependencies: map[string]Dependency{
					VMConfigDepName: {
						Version: "dev",
						URL:     "",
						Type:    "tar.gz",
						Digest:  "",
					},
				},
			}}
			if err := json.NewEncoder(w).Encode(manifest); err != nil {
				t.Errorf("encode manifest: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	manager := NewManager(server.URL, t.TempDir())
	err := manager.Download(DefaultKind, "arm64", DownloadOptions{DryRun: true, Client: server.Client()})
	if err == nil {
		t.Fatal("Download returned nil error, want unreleased vmconfig stub error")
	}
	if !strings.Contains(err.Error(), "published vmconfig manifest contains unreleased vmconfig stub") {
		t.Fatalf("Download error = %q, want unreleased vmconfig stub", err.Error())
	}
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func serverURL(r *http.Request, path string) string {
	return "http://" + r.Host + path
}
