// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package deps

import "strings"

const (
	TemplateChartTypeOCI   = "oci"
	TemplateChartTypeChart = "chart"
)

// TemplatesManifest is the top-level templates dependency manifest.
type TemplatesManifest struct {
	Templates Templates `json:"templates"`
}

type Templates struct {
	Version string          `json:"version"`
	Charts  []TemplateChart `json:"charts"`
}

type TemplateChart struct {
	Name             string               `json:"name"`
	Version          string               `json:"version"`
	Type             string               `json:"type"`
	URL              string               `json:"url,omitempty"`
	Path             string               `json:"path,omitempty"`
	Digest           string               `json:"digest,omitempty"`
	Dependencies     TemplateDependencies `json:"dependencies,omitempty"`
	Cached           bool                 `json:"cached,omitempty"`
	ChartLayerDigest string               `json:"chartLayerDigest,omitempty"`
}

type TemplateDependencies struct {
	Components []string `json:"components,omitempty"`
}

// ResetCached clears local cache-state markers from template charts.
func (m *TemplatesManifest) ResetCached() {
	if m == nil {
		return
	}
	for i := range m.Templates.Charts {
		m.Templates.Charts[i].Cached = false
		m.Templates.Charts[i].ChartLayerDigest = ""
	}
}

// MarkCached marks a template chart as cached with its chart layer digest.
func (m *TemplatesManifest) MarkCached(index int, chartLayerDigest string) {
	if m == nil || index < 0 || index >= len(m.Templates.Charts) {
		return
	}
	m.Templates.Charts[index].Cached = true
	m.Templates.Charts[index].ChartLayerDigest = chartLayerDigest
}

// helmChartTag returns the OCI tag form Helm uses for a chart version.
func helmChartTag(version string) string {
	return strings.ReplaceAll(version, "+", "_")
}
