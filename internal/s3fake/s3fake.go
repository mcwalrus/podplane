// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

// Package s3fake exposes a lightweight, on-disk fake S3 server suitable for
// embedding in the local Podplane background server.
//
// It wraps github.com/johannesboyne/gofakes3 with the s3afero MultiBucket
// backend so tests / dev clusters can read and write a normal filesystem
// directory as if it were S3.
package s3fake

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3afero"
	"github.com/spf13/afero"
)

// Handler returns an S3-compatible multi-bucket API backed by the supplied
// storage directory. Buckets are created on first use.
//
// storageDir must exist (or be creatable). If it does not exist it is created
// with mode 0700.
func Handler(storageDir string) (http.Handler, error) {
	if storageDir == "" {
		return nil, fmt.Errorf("s3fake: storageDir is required")
	}
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("s3fake: mkdir storage: %w", err)
	}

	baseFs := afero.NewBasePathFs(afero.NewOsFs(), storageDir)
	backend, err := s3afero.MultiBucket(baseFs)
	if err != nil {
		return nil, fmt.Errorf("s3fake: build backend: %w", err)
	}
	faker := gofakes3.New(
		backend,
		gofakes3.WithAutoBucket(true),
		gofakes3.WithLogger(&renamingLogger{inner: gofakes3.GlobalLog()}),
	)
	return faker.Server(), nil
}

// BucketHandler returns an S3-compatible single-bucket API backed by storageDir.
func BucketHandler(bucketName, storageDir string) (http.Handler, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("s3fake: bucketName is required")
	}
	if storageDir == "" {
		return nil, fmt.Errorf("s3fake: storageDir is required")
	}
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("s3fake: mkdir storage: %w", err)
	}

	baseFs := afero.NewBasePathFs(afero.NewOsFs(), storageDir)
	backend, err := s3afero.SingleBucket(bucketName, baseFs, afero.NewMemMapFs())
	if err != nil {
		return nil, fmt.Errorf("s3fake: build bucket backend: %w", err)
	}
	faker := gofakes3.New(
		backend,
		gofakes3.WithLogger(&renamingLogger{inner: gofakes3.GlobalLog()}),
	)
	return faker.Server(), nil
}

// renamingLogger wraps a gofakes3.Logger to rewrite the slightly misleading
// "CREATE OBJECT" log message that the library prints for object PUTs into
// the more familiar "PUT OBJECT" so log output reads naturally.
type renamingLogger struct {
	inner gofakes3.Logger
}

func (l *renamingLogger) Print(level gofakes3.LogLevel, v ...any) {
	rewritten := make([]any, len(v))
	for i, item := range v {
		if s, ok := item.(string); ok {
			rewritten[i] = strings.ReplaceAll(s, "CREATE OBJECT", "PUT OBJECT")
		} else {
			rewritten[i] = item
		}
	}
	l.inner.Print(level, rewritten...)
}
