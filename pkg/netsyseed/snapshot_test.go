// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package netsyseed

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/podplane/internal/clusterconfig"
)

func TestInterpolatePlatformComponentsMergesValues(t *testing.T) {
	records := []*datafile.Record{
		{Key: []byte("/registry/configmaps/default/ignored"), Value: []byte(`{"kind":"ConfigMap","metadata":{"name":"ignored"}}`)},
		{Key: []byte(platformComponentsHelmReleaseKey), Value: []byte(`{"apiVersion":"helm.toolkit.fluxcd.io/v2","kind":"HelmRelease","metadata":{"name":"platform-components","namespace":"platform-components"},"spec":{"values":{"platform":{"components":{"apps":{"cilium":{"enabled":true}}}}}}}`)},
	}
	values := map[string]any{
		"platform": map[string]any{
			"components": map[string]any{
				"apps": map[string]any{
					"traefik": map[string]any{"enabled": true},
				},
			},
		},
	}

	if err := interpolatePlatformComponents(records, values); err != nil {
		t.Fatalf("interpolatePlatformComponents error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(records[1].Value, &got); err != nil {
		t.Fatalf("unmarshal interpolated HelmRelease: %v", err)
	}
	apps := got["spec"].(map[string]any)["values"].(map[string]any)["platform"].(map[string]any)["components"].(map[string]any)["apps"].(map[string]any)
	if apps["cilium"] == nil {
		t.Fatalf("existing app values were not preserved")
	}
	if apps["traefik"] == nil {
		t.Fatalf("derived app values were not merged")
	}
}

func TestInterpolateComponentsSourceUpdatesGitRepository(t *testing.T) {
	records := []*datafile.Record{
		{Key: []byte(platformComponentsHelmReleaseKey), Value: []byte(`{"kind":"HelmRelease","metadata":{"name":"ignored"}}`)},
		{Key: []byte(podplaneComponentsGitKey), Value: []byte(`{"apiVersion":"source.toolkit.fluxcd.io/v1","kind":"GitRepository","metadata":{"name":"podplane-components","namespace":"platform-components"},"spec":{"url":"https://github.com/podplane/components.git","ref":{"branch":"main"}}}`)},
	}
	source := &clusterconfig.ComponentsSource{
		URL: "https://github.com/example/components.git",
		Ref: clusterconfig.ComponentsSourceRef{Branch: "feature"},
	}

	if err := interpolateComponentsSource(records, source); err != nil {
		t.Fatalf("interpolateComponentsSource error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(records[1].Value, &got); err != nil {
		t.Fatalf("unmarshal interpolated GitRepository: %v", err)
	}
	spec := got["spec"].(map[string]any)
	if got, want := spec["url"], "https://github.com/example/components.git"; got != want {
		t.Fatalf("spec.url = %v, want %v", got, want)
	}
	ref := spec["ref"].(map[string]any)
	if got, want := ref["branch"], "feature"; got != want {
		t.Fatalf("spec.ref.branch = %v, want %v", got, want)
	}
}

func TestMergeValuesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(path, []byte(`
platform:
  components:
    registry:
      mirror:
        enabled: true
        hostname: first.example.com
    apps:
      traefik:
        enabled: true
`), 0o600); err != nil {
		t.Fatalf("write values: %v", err)
	}
	values := map[string]any{
		"platform": map[string]any{
			"components": map[string]any{
				"apps": map[string]any{
					"cilium": map[string]any{"enabled": true},
				},
			},
		},
	}

	if err := mergeValuesFile(values, path); err != nil {
		t.Fatalf("mergeValuesFile error = %v", err)
	}
	components := values["platform"].(map[string]any)["components"].(map[string]any)
	mirror := components["registry"].(map[string]any)["mirror"].(map[string]any)
	if got, want := mirror["enabled"], true; got != want {
		t.Fatalf("mirror.enabled = %v, want %v", got, want)
	}
	if got, want := mirror["hostname"], "first.example.com"; got != want {
		t.Fatalf("mirror.hostname = %v, want %v", got, want)
	}
	apps := components["apps"].(map[string]any)
	if apps["cilium"] == nil || apps["traefik"] == nil {
		t.Fatalf("apps were not merged: %v", apps)
	}
}

// TestWriteSnapshotWritesBytes verifies WriteSnapshot writes the rendered
// snapshot bytes from an explicit seed file.
func TestWriteSnapshotWritesBytes(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &clusterconfig.ClusterConfig{Cluster: clusterconfig.Cluster{
		ID:   "local",
		OIDC: clusterconfig.OIDC{IssuerURL: "https://oidc.localhost/oidc"},
		Domains: []clusterconfig.Domain{
			{Zone: "local.localhost", Provider: clusterconfig.DomainProvider{Kind: "local"}},
		},
	}}
	// Patch validation by re-marshalling and using a non-reserved ID for Load().
	cfg.Cluster.ID = "localdev"
	cfgPath := filepath.Join(tmpDir, "cluster.jsonc")
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal cluster cfg: %v", err)
	}
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatalf("write cluster cfg: %v", err)
	}

	// Build a tiny seed snapshot in memory that contains the
	// platform-components HelmRelease record at the expected key. Serve it
	// over HTTP so loadSeedFile's URL path is exercised.
	records := []*datafile.Record{
		{Revision: 5, Key: []byte(platformComponentsHelmReleaseKey), Value: []byte(`{"apiVersion":"helm.toolkit.fluxcd.io/v2","kind":"HelmRelease","metadata":{"name":"platform-components","namespace":"platform-components"},"spec":{"values":{}}}`)},
	}
	var buf bytes.Buffer
	if err := datafile.WriteSnapshot(&buf, records, "localdev"); err != nil {
		t.Fatalf("write seed snapshot: %v", err)
	}
	seedBytes := buf.Bytes()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/recommended.netsy") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(seedBytes)
	}))
	defer server.Close()

	var data bytes.Buffer
	if err := WriteSnapshot(&data, SnapshotOptions{
		ClusterConfigPath: cfgPath,
		SeedPath:          server.URL + "/recommended.netsy",
	}); err != nil {
		t.Fatalf("WriteSnapshot error = %v", err)
	}
	if data.Len() == 0 {
		t.Fatalf("WriteSnapshot wrote no data")
	}

	// Verify the interpolated values contain our seeded traefik domain.
	got, err := datafile.ReadSnapshot(bytes.NewReader(data.Bytes()))
	if err != nil {
		t.Fatalf("read built snapshot: %v", err)
	}
	var found bool
	for _, r := range got {
		if string(r.Key) != platformComponentsHelmReleaseKey {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(r.Value, &obj); err != nil {
			t.Fatalf("unmarshal HelmRelease: %v", err)
		}
		traefik := obj["spec"].(map[string]any)["values"].(map[string]any)["platform"].(map[string]any)["components"].(map[string]any)["values"].(map[string]any)["traefik"].(map[string]any)
		zone := traefik["platform"].(map[string]any)["traefik"].(map[string]any)["ingress"].(map[string]any)["domains"].([]any)[0].(map[string]any)["zone"]
		if zone != "local.localhost" {
			t.Fatalf("seeded ingress zone = %v, want local.localhost", zone)
		}
		found = true
	}
	if !found {
		t.Fatalf("platform-components HelmRelease not found in built snapshot")
	}
}
