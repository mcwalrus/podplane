// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package registryclient

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// remoteRef resolves and validates the target registry image reference.
func remoteRef(registryHost string, source name.Reference, remoteImage string) (name.Reference, error) {
	if remoteImage == "" {
		repo := source.Context().RepositoryStr()
		if i := strings.LastIndex(repo, "/"); i >= 0 {
			repo = repo[i+1:]
		}
		remoteImage = registryHost + "/apps/" + repo + ":" + source.Identifier()
	} else if !imageHasRegistry(remoteImage) {
		remoteImage = registryHost + "/" + strings.TrimPrefix(remoteImage, "/")
	}
	ref, err := name.ParseReference(remoteImage, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("parse remote image %q: %w", remoteImage, err)
	}
	repo := ref.Context().RepositoryStr()
	if !strings.HasPrefix(repo, "apps/") && repo != "apps" {
		return nil, fmt.Errorf("remote image %q must be under apps/**; mirror/** is reserved for Podplane-managed dependency images", ref.Name())
	}
	return ref, nil
}

// imageHasRegistry reports whether an image reference starts with an explicit registry host.
func imageHasRegistry(ref string) bool {
	slash := strings.Index(ref, "/")
	if slash < 0 {
		return false
	}
	first := ref[:slash]
	return strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost"
}
