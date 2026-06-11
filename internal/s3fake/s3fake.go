// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

// Package s3fake exposes a lightweight, on-disk fake S3 server suitable for
// embedding in the local Podplane background server.
//
// It wraps github.com/johannesboyne/gofakes3 with the s3afero MultiBucket
// backend so tests / dev clusters can read and write a normal filesystem
// directory as if it were S3.
package s3fake

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

// WriteObject writes one object directly to the fake S3 on-disk storage using
// the same gofakes3 backend as Handler. It is useful for seeding local buckets
// before the HTTP server is running while still preserving backend metadata.
func WriteObject(storageDir, bucketName, objectName string, data []byte) error {
	if storageDir == "" {
		return fmt.Errorf("s3fake: storageDir is required")
	}
	if bucketName == "" {
		return fmt.Errorf("s3fake: bucketName is required")
	}
	if objectName == "" {
		return fmt.Errorf("s3fake: objectName is required")
	}
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return fmt.Errorf("s3fake: mkdir storage: %w", err)
	}

	baseFs := afero.NewBasePathFs(afero.NewOsFs(), storageDir)
	backend, err := s3afero.MultiBucket(baseFs)
	if err != nil {
		return fmt.Errorf("s3fake: build backend: %w", err)
	}
	if err := backend.CreateBucket(bucketName); err != nil && !gofakes3.IsAlreadyExists(err) {
		return fmt.Errorf("s3fake: create bucket %s: %w", bucketName, err)
	}
	if _, err := backend.PutObject(bucketName, objectName, nil, bytes.NewReader(data), int64(len(data)), nil); err != nil {
		return fmt.Errorf("s3fake: write object %s/%s: %w", bucketName, objectName, err)
	}
	return nil
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
	wrappedBackend := &cacheBucketBackend{Backend: backend, fs: baseFs}
	faker := gofakes3.New(
		wrappedBackend,
		gofakes3.WithLogger(&renamingLogger{inner: gofakes3.GlobalLog()}),
	)
	return faker.Server(), nil
}

// cacheBucketBackend adapts gofakes3's filesystem single-bucket backend for
// cache directories that are populated directly on disk rather than via S3 PUT.
type cacheBucketBackend struct {
	gofakes3.Backend
	fs afero.Fs
}

// cacheBucketListItem is one sortable result item from a ListBucket response.
type cacheBucketListItem struct {
	key     string
	content *gofakes3.Content
	prefix  *gofakes3.CommonPrefix
}

// ListBucket adds paging support to s3afero.SingleBucket, which otherwise asks
// gofakes3 to retry list requests without max-keys/marker handling.
func (b *cacheBucketBackend) ListBucket(name string, prefix *gofakes3.Prefix, page gofakes3.ListBucketPage) (*gofakes3.ObjectList, error) {
	objects, err := b.Backend.ListBucket(name, prefix, gofakes3.ListBucketPage{})
	if err != nil {
		return nil, err
	}
	return applyListBucketPage(objects, page), nil
}

// HeadObject ensures filesystem-backed objects have Last-Modified metadata.
func (b *cacheBucketBackend) HeadObject(bucketName, objectName string) (*gofakes3.Object, error) {
	object, err := b.Backend.HeadObject(bucketName, objectName)
	if err != nil {
		return nil, err
	}
	b.decorateObjectMetadata(object)
	return object, nil
}

// GetObject ensures filesystem-backed objects have Last-Modified metadata.
func (b *cacheBucketBackend) GetObject(bucketName, objectName string, rangeRequest *gofakes3.ObjectRangeRequest) (*gofakes3.Object, error) {
	object, err := b.Backend.GetObject(bucketName, objectName, rangeRequest)
	if err != nil {
		return nil, err
	}
	b.decorateObjectMetadata(object)
	return object, nil
}

// decorateObjectMetadata backfills Last-Modified for direct filesystem files.
func (b *cacheBucketBackend) decorateObjectMetadata(object *gofakes3.Object) {
	if object == nil || object.Metadata["Last-Modified"] != "" {
		return
	}
	if object.Metadata == nil {
		object.Metadata = map[string]string{}
	}
	stat, err := b.fs.Stat(filepath.FromSlash(object.Name))
	if err != nil {
		return
	}
	object.Metadata["Last-Modified"] = stat.ModTime().UTC().Format(http.TimeFormat)
}

// applyListBucketPage applies marker/max-keys pagination to a full object list.
func applyListBucketPage(objects *gofakes3.ObjectList, page gofakes3.ListBucketPage) *gofakes3.ObjectList {
	if objects == nil || page.IsEmpty() {
		return objects
	}
	items := make([]cacheBucketListItem, 0, len(objects.Contents)+len(objects.CommonPrefixes))
	for _, content := range objects.Contents {
		if content != nil {
			items = append(items, cacheBucketListItem{key: content.Key, content: content})
		}
	}
	for i := range objects.CommonPrefixes {
		prefix := objects.CommonPrefixes[i]
		items = append(items, cacheBucketListItem{key: prefix.Prefix, prefix: &prefix})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })

	paged := gofakes3.NewObjectList()
	var count int64
	for _, item := range items {
		if page.HasMarker && item.key <= page.Marker {
			continue
		}
		if page.MaxKeys > 0 && count >= page.MaxKeys {
			paged.IsTruncated = true
			break
		}
		if item.content != nil {
			paged.Add(item.content)
		} else if item.prefix != nil {
			paged.AddPrefix(item.prefix.Prefix)
		}
		paged.NextMarker = item.key
		count++
	}
	return paged
}

// renamingLogger wraps a gofakes3.Logger to enable log rewriting.
type renamingLogger struct {
	inner gofakes3.Logger
}

// Print rewrites logs before forwarding to the underlying gofakes3 logger.
// Currently it:
//   - Rewrites the slightly misleading "CREATE OBJECT" log message that the
//     library prints for object PUTs into the more familiar "PUT OBJECT".
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
