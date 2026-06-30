// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package registryclient

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

func TestLocalDockerImageNameAddsParsedDefaultTag(t *testing.T) {
	tag := mustTag(t, "example-api:latest")
	if got, want := localDockerImageName("example-api", tag), "example-api:latest"; got != want {
		t.Fatalf("localDockerImageName() = %q, want %q", got, want)
	}
}

func TestLocalDockerImageNamePreservesExplicitTag(t *testing.T) {
	tag := mustTag(t, "registry.example.com/acme/example-api:v1")
	if got, want := localDockerImageName("registry.example.com/acme/example-api:v1", tag), "registry.example.com/acme/example-api:v1"; got != want {
		t.Fatalf("localDockerImageName() = %q, want %q", got, want)
	}
}

func TestLocalDockerImageNamePreservesDigest(t *testing.T) {
	input := "registry.example.com/acme/example-api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tag := mustTag(t, "registry.example.com/acme/example-api:latest")
	if got := localDockerImageName(input, tag); got != input {
		t.Fatalf("localDockerImageName() = %q, want %q", got, input)
	}
}

func mustTag(t *testing.T, ref string) name.Tag {
	t.Helper()
	tag, err := name.NewTag(ref, name.WeakValidation)
	if err != nil {
		t.Fatal(err)
	}
	return tag
}
