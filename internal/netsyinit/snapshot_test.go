// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package netsyinit

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	if err := InterpolatePlatformComponents(records, values); err != nil {
		t.Fatalf("InterpolatePlatformComponents error = %v", err)
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

	if err := InterpolateComponentsSource(records, source); err != nil {
		t.Fatalf("InterpolateComponentsSource error = %v", err)
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

	if err := MergeValuesFile(values, path); err != nil {
		t.Fatalf("MergeValuesFile error = %v", err)
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
