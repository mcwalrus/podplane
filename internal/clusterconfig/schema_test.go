// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package clusterconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClusterSchemaJSONIsValidJSON(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(ClusterSchemaJSON), &schema); err != nil {
		t.Fatalf("ClusterSchemaJSON is invalid JSON: %v", err)
	}
	if got := schema["title"]; got != "Podplane cluster config" {
		t.Fatalf("schema title = %v, want Podplane cluster config", got)
	}
}

func TestWriteAddsDefaultLocalSchemaRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "podplane.cluster.jsonc")
	cfg := NewDraftConfig("aws")
	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got ClusterConfig
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("written config is invalid JSON: %v", err)
	}
	if got.Schema != DefaultSchemaRef {
		t.Fatalf("written schema ref = %q, want %q", got.Schema, DefaultSchemaRef)
	}
	if cfg.Schema != "" {
		t.Fatalf("Write mutated input schema ref to %q", cfg.Schema)
	}
}

func TestWriteSchemaWritesLocalSchemaFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSchema(dir); err != nil {
		t.Fatalf("WriteSchema returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, SchemaFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != ClusterSchemaJSON {
		t.Fatal("schema file content did not match ClusterSchemaJSON")
	}
}
