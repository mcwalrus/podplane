// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package osboot

import (
	"reflect"
	"testing"
)

func TestFilesystemPathsIncludesBootPartitionRelativeFallback(t *testing.T) {
	got := filesystemPaths("/boot/vmlinuz-6.12.85+deb13-cloud-arm64")
	want := []string{"boot/vmlinuz-6.12.85+deb13-cloud-arm64", "vmlinuz-6.12.85+deb13-cloud-arm64"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filesystemPaths() = %#v, want %#v", got, want)
	}
}

func TestFilesystemPathsRejectsEmptyPath(t *testing.T) {
	if got := filesystemPaths("/"); len(got) != 0 {
		t.Fatalf("filesystemPaths() = %#v, want empty", got)
	}
}
