// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package oidcconfig

import (
	"fmt"
	"os"
	"path/filepath"

	schemaassets "github.com/podplane/podplane/schemas"
)

// SchemaFileName is the local JSON Schema file written next to OIDC configs.
const SchemaFileName = "podplane.oidc.schema.json"

// DefaultSchemaRef is the relative schema reference embedded in new OIDC configs.
const DefaultSchemaRef = "./" + SchemaFileName

// SchemaJSON is the JSON Schema for podplane.oidc.jsonc files.
var SchemaJSON = schemaassets.OIDCSchemaJSON

// WriteSchema writes the local JSON Schema file used by editors for offline
// validation, completion, and hover documentation.
func WriteSchema(dir string) error {
	path := filepath.Join(dir, SchemaFileName)
	if err := os.WriteFile(path, []byte(SchemaJSON), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
