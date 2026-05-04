// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

// ConfigSource is the subset of CLI config needed to build local options.
type ConfigSource interface {
	ConfigDirectory() string
	CacheDirectory() string
	DataDirectory() string
	RuntimeDirectory() string
	DepsBaseURL() string
	DepsCacheDir() string
	InstanceKind() string
	Arch() string
}
