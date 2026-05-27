// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package deploy

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTemplateValuesSchemaRequiresSchema(t *testing.T) {
	t.Parallel()
	err := validateTemplateValuesSchema("worker", t.TempDir(), false, false)
	if err == nil {
		t.Fatal("expected missing schema error")
	}
}

func TestValidateTemplateValuesSchemaRejectsUnsupportedRouteFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSchema(t, dir, `{"properties":{"image":{}}}`)
	err := validateTemplateValuesSchema("worker", dir, true, false)
	if err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestValidateTemplateValuesSchemaAcceptsSupportedRouteFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeSchema(t, dir, `{"properties":{"route":{"properties":{"hostname":{},"path":{}}}}}`)
	if err := validateTemplateValuesSchema("web", dir, true, true); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTemplateValuesSchemaAcceptsChartArchive(t *testing.T) {
	t.Parallel()
	chartPath := filepath.Join(t.TempDir(), "web-1.0.0.tgz")
	writeChartArchive(t, chartPath, `{"properties":{"route":{"properties":{"hostname":{},"path":{}}}}}`)
	if err := validateTemplateValuesSchema("web", chartPath, true, true); err != nil {
		t.Fatal(err)
	}
}

func writeSchema(t *testing.T, dir, schema string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, valuesSchemaFile), []byte(schema), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeChartArchive(t *testing.T, chartPath, schema string) {
	t.Helper()
	f, err := os.Create(chartPath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	contents := []byte(schema)
	if err := tw.WriteHeader(&tar.Header{
		Name: "web/" + valuesSchemaFile,
		Mode: 0o600,
		Size: int64(len(contents)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
