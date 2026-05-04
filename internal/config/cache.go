// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/podplane/podplane/internal/filecache"
)

// CachePath returns the absolute path of a file or directory under the cache
// directory (i.e. ~/.podplane/cache/something)
func (c *Config) CachePath(opts ...string) (string, error) {
	// get the cache directory
	cacheDir := c.CacheDirectory()
	// get the full path including opts
	fullPath := cacheDir
	if len(opts) > 0 {
		fullPath = filepath.Join(append([]string{cacheDir}, opts...)...)
	}
	// create the dir(s) in the full path if required
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if err := os.MkdirAll(fullPath, 0777); err != nil {
			return "", fmt.Errorf("failed to create cache directory %s: %w", fullPath, err)
		}
	}
	return fullPath, nil
}

// ResolveCACert resolves a CA cert spec (inline PEM / http(s) URL / file
// path) to an absolute path on disk, caching URL fetches and inline PEMs
// under <CacheDirectory>/<subdir>. An empty spec returns ("", nil).
func (c *Config) ResolveCACert(subdir, spec string) (string, error) {
	if spec == "" {
		return "", nil
	}
	cacheDir, err := c.CachePath(subdir)
	if err != nil {
		return "", err
	}
	return filecache.ResolveCACert(spec, cacheDir)
}
