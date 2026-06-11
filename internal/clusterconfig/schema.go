// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package clusterconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/podplane/podplane/schemas"
)

// SchemaFileName is the local JSON Schema file written next to cluster configs.
const SchemaFileName = "podplane.cluster.schema.json"

// DefaultSchemaRef is the relative schema reference embedded in new cluster configs.
const DefaultSchemaRef = "./" + SchemaFileName

// ClusterSchemaJSON is the JSON Schema for podplane.cluster.jsonc files.
var ClusterSchemaJSON = schemas.ClusterSchemaJSON

// WriteSchema writes the local JSON Schema file used by editors for offline
// validation, completion, and hover documentation.
func WriteSchema(dir string) error {
	path := filepath.Join(dir, SchemaFileName)
	content := schemas.GeneratedSchemaCopy(ClusterSchemaJSON, "schemas/podplane.cluster.schema.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
