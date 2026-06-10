// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

// Package schemas embeds Podplane JSON Schemas for offline CLI output.
package schemas

import _ "embed"

// ClusterSchemaJSON is the JSON Schema for podplane.cluster.jsonc files.
//
//go:embed podplane.cluster.schema.json
var ClusterSchemaJSON string
