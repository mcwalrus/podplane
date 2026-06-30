// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package registryclient

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/podplane/podplane/internal/execwrap"
)

// localImage exports a local Docker image to a temporary tarball and reads it as an OCI image.
func localImage(sourceInput string, source name.Reference) (v1.Image, func(), error) {
	tag, ok := source.(name.Tag)
	if !ok {
		return nil, func() {}, fmt.Errorf("local source image must be a tag, got %s", source.Name())
	}
	tmp, err := os.CreateTemp("", "podplane-image-*.tar")
	if err != nil {
		return nil, func() {}, err
	}
	path := tmp.Name()
	cleanup := func() { _ = os.Remove(path) }
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	saveName := localDockerImageName(sourceInput, tag)
	save := execwrap.Command("docker", "image", "save", "--output", path, saveName)
	var stderr bytes.Buffer
	save.Stderr = &stderr
	if err := save.Run(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("docker image save: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	saveTag, err := name.NewTag(saveName, name.WeakValidation)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	img, err := tarball.ImageFromPath(path, &saveTag)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return img, cleanup, nil
}

// localDockerImageName returns a Docker CLI image name, adding the parsed default tag when omitted.
func localDockerImageName(sourceInput string, tag name.Tag) string {
	lastSlash := strings.LastIndex(sourceInput, "/")
	lastColon := strings.LastIndex(sourceInput, ":")
	if strings.Contains(sourceInput, "@") || lastColon > lastSlash {
		return sourceInput
	}
	return sourceInput + ":" + tag.Identifier()
}
