// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package netsyinit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/netsy-dev/netsy/pkg/datafile"
	"github.com/podplane/podplane/internal/clusterconfig"
	"gopkg.in/yaml.v3"
)

const (
	podplaneComponentsGitKey         = "/registry/source.toolkit.fluxcd.io/gitrepositories/platform-components/podplane-components"
	platformComponentsHelmReleaseKey = "/registry/helm.toolkit.fluxcd.io/helmreleases/platform-components/platform-components"
)

type SnapshotOptions struct {
	ClusterConfigPath string
	Template          string
	DepsBaseURL       string
	ValuesFile        string
}

// WriteSnapshot writes an initial Netsy snapshot with platform-components
// values derived from the cluster config interpolated into the template.
func WriteSnapshot(w io.Writer, opts SnapshotOptions) error {
	cluster, err := clusterconfig.Load(opts.ClusterConfigPath)
	if err != nil {
		return err
	}
	values, err := BuildPlatformComponentsValues(cluster)
	if err != nil {
		return err
	}
	if err := MergeValuesFile(values, opts.ValuesFile); err != nil {
		return err
	}
	templateData, err := LoadTemplate(opts)
	if err != nil {
		return err
	}
	records, err := datafile.ReadSnapshot(bytes.NewReader(templateData))
	if err != nil {
		return fmt.Errorf("read Netsy snapshot template: %w", err)
	}
	if err := InterpolatePlatformComponents(records, values); err != nil {
		return err
	}
	if err := InterpolateComponentsSource(records, cluster.Cluster.Components.Source); err != nil {
		return err
	}
	if err := datafile.WriteSnapshot(w, records, cluster.Cluster.ID); err != nil {
		return fmt.Errorf("write Netsy snapshot: %w", err)
	}
	return nil
}

// MergeValuesFile merges a YAML/JSON values file over dst.
func MergeValuesFile(dst map[string]any, path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read values file %s: %w", path, err)
	}
	var values map[string]any
	if err := yaml.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("decode values file %s: %w", path, err)
	}
	if values != nil {
		deepMerge(dst, values)
	}
	return nil
}

// LoadTemplate returns the Netsy snapshot template from an explicit local path
// or URL when provided, otherwise downloading it from the default URL.
func LoadTemplate(opts SnapshotOptions) ([]byte, error) {
	template := opts.Template
	if template != "" {
		parsed, err := url.Parse(template)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			data, err := os.ReadFile(template)
			if err != nil {
				return nil, fmt.Errorf("read Netsy snapshot template %s: %w", template, err)
			}
			return data, nil
		}
	} else {
		template = DefaultTemplateURL(opts.DepsBaseURL)
	}
	resp, err := http.Get(template)
	if err != nil {
		return nil, fmt.Errorf("download Netsy snapshot template %s: %w", template, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download Netsy snapshot template %s: HTTP %s", template, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Netsy snapshot template response: %w", err)
	}
	return data, nil
}

// DefaultTemplateURL builds the default Netsy snapshot template URL from the
// dependency base URL.
func DefaultTemplateURL(depsBaseURL string) string {
	return strings.TrimRight(depsBaseURL, "/") + "/netsy/recommended.netsy"
}

// InterpolatePlatformComponents merges derived platform-components values into
// the platform-components HelmRelease record in a Netsy snapshot.
func InterpolatePlatformComponents(records []*datafile.Record, values map[string]any) error {
	for i := range records {
		if string(records[i].Key) != platformComponentsHelmReleaseKey {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(records[i].Value, &obj); err != nil {
			return fmt.Errorf("decode platform-components HelmRelease at %s: %w", platformComponentsHelmReleaseKey, err)
		}
		if obj["kind"] != "HelmRelease" || !strings.HasPrefix(stringValue(obj["apiVersion"]), "helm.toolkit.fluxcd.io/") {
			return fmt.Errorf("record at %s is not a Flux HelmRelease", platformComponentsHelmReleaseKey)
		}
		metadata, _ := obj["metadata"].(map[string]any)
		if metadata["name"] != "platform-components" {
			return fmt.Errorf("record at %s is not the platform-components HelmRelease", platformComponentsHelmReleaseKey)
		}
		if namespace := stringValue(metadata["namespace"]); namespace != "" && namespace != "platform-components" {
			return fmt.Errorf("record at %s is in namespace %q, want platform-components", platformComponentsHelmReleaseKey, namespace)
		}
		spec := ensureMap(obj, "spec")
		specValues := ensureMap(spec, "values")
		deepMerge(specValues, values)
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(obj); err != nil {
			return fmt.Errorf("encode platform-components HelmRelease: %w", err)
		}
		records[i].Value = buf.Bytes()
		return nil
	}
	return fmt.Errorf("Netsy snapshot template does not contain the platform-components HelmRelease at %s", platformComponentsHelmReleaseKey)
}

// InterpolateComponentsSource updates the bootstrap GitRepository used by Flux
// to source the platform-components chart and child component HelmReleases.
func InterpolateComponentsSource(records []*datafile.Record, source *clusterconfig.ComponentsSource) error {
	if source == nil || source.URL == "" {
		return nil
	}
	for i := range records {
		if string(records[i].Key) != podplaneComponentsGitKey {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(records[i].Value, &obj); err != nil {
			return fmt.Errorf("decode podplane-components GitRepository at %s: %w", podplaneComponentsGitKey, err)
		}
		if obj["kind"] != "GitRepository" || !strings.HasPrefix(stringValue(obj["apiVersion"]), "source.toolkit.fluxcd.io/") {
			return fmt.Errorf("record at %s is not a Flux GitRepository", podplaneComponentsGitKey)
		}
		metadata, _ := obj["metadata"].(map[string]any)
		if metadata["name"] != "podplane-components" {
			return fmt.Errorf("record at %s is not the podplane-components GitRepository", podplaneComponentsGitKey)
		}
		if namespace := stringValue(metadata["namespace"]); namespace != "" && namespace != "platform-components" {
			return fmt.Errorf("record at %s is in namespace %q, want platform-components", podplaneComponentsGitKey, namespace)
		}
		spec := ensureMap(obj, "spec")
		spec["url"] = source.URL
		ref := map[string]any{}
		if source.Ref.Branch != "" {
			ref["branch"] = source.Ref.Branch
		}
		if source.Ref.Tag != "" {
			ref["tag"] = source.Ref.Tag
		}
		if source.Ref.Semver != "" {
			ref["semver"] = source.Ref.Semver
		}
		if source.Ref.Commit != "" {
			ref["commit"] = source.Ref.Commit
		}
		if len(ref) > 0 {
			spec["ref"] = ref
		} else {
			delete(spec, "ref")
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(obj); err != nil {
			return fmt.Errorf("encode podplane-components GitRepository: %w", err)
		}
		records[i].Value = buf.Bytes()
		return nil
	}
	return fmt.Errorf("Netsy snapshot template does not contain the podplane-components GitRepository at %s", podplaneComponentsGitKey)
}

// ensureMap returns the existing map value for key or creates and stores a new
// map when the key is absent or not already a map.
func ensureMap(parent map[string]any, key string) map[string]any {
	if child, ok := parent[key].(map[string]any); ok {
		return child
	}
	child := map[string]any{}
	parent[key] = child
	return child
}

// deepMerge recursively merges src into dst, preserving existing nested maps
// and replacing non-map values.
func deepMerge(dst, src map[string]any) {
	for key, value := range src {
		if srcMap, ok := value.(map[string]any); ok {
			if dstMap, ok := dst[key].(map[string]any); ok {
				deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[key] = value
	}
}

// stringValue returns value when it is a string, otherwise returning the empty
// string.
func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
