// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/deps"
)

// Chart is a resolved template chart in the local dependency cache.
type Chart struct {
	Path     string
	Template deps.TemplateChart
}

// EnsureChart returns the local Helm chart path for template, downloading
// dependencies first when the chart is missing from the cache. wrap, if
// non-nil, brackets the download call so callers can render a UI around it;
// it receives the download function to invoke with a progress callback.
func EnsureChart(c *config.Config, template string, wrap func(download func(progress func(deps.DownloadEvent)) error) error) (Chart, error) {
	m := deps.NewManager(c.DepsBaseURL(), c.DepsCacheDir())
	if chart, path, err := m.CachedTemplateChart(template); err == nil {
		return Chart{Path: path, Template: chart}, nil
	}
	download := func(progress func(deps.DownloadEvent)) error {
		return m.Download(c.InstanceKind(), c.Arch(), deps.DownloadOptions{Progress: progress})
	}
	if wrap != nil {
		if err := wrap(download); err != nil {
			return Chart{}, err
		}
	} else if err := download(nil); err != nil {
		return Chart{}, err
	}
	chart, path, err := m.CachedTemplateChart(template)
	if err != nil {
		return Chart{}, err
	}
	return Chart{Path: path, Template: chart}, nil
}
