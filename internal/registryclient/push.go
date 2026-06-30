// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package registryclient

import (
	"context"
	"fmt"
	"io"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/podplane/podplane/internal/config"
	"github.com/podplane/podplane/internal/kubectl"
)

// Options configures a registry image push.
type Options struct {
	Config     *config.Config
	Source     string
	Remote     string
	Context    string
	Kubeconfig string
	Stderr     io.Writer
}

// Push pushes a local image to the current Podplane cluster registry.
func Push(ctx context.Context, opts Options) (string, error) {
	if opts.Config == nil {
		return "", fmt.Errorf("config is required")
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	clusterID, local, err := kubectl.ClusterIDFromContext(opts.Context, opts.Kubeconfig)
	if err != nil {
		return "", err
	}
	summary, err := opts.Config.ClusterSummary(clusterID, local)
	if err != nil {
		return "", err
	}
	if summary.ID == "" {
		return "", fmt.Errorf("cluster summary for %q is not cached; run `podplane login -f <podplane.cluster.jsonc>` for this cluster", clusterID)
	}
	if summary.Registry.Hostname == "" {
		return "", fmt.Errorf("cluster %q has no cluster.registry.hostname configured; rerun login/local start after configuring the registry hostname", clusterID)
	}
	if err := ensureZotRegistryReady(opts.Context, opts.Kubeconfig); err != nil {
		return "", err
	}

	source, err := name.ParseReference(opts.Source, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("parse local image %q: %w", opts.Source, err)
	}
	remoteRef, err := remoteRef(summary.Registry.Hostname, source, opts.Remote)
	if err != nil {
		return "", err
	}
	token, err := resolvePushToken(opts.Config, clusterID, local, opts.Context, opts.Kubeconfig)
	if err != nil {
		return "", err
	}

	img, cleanup, err := localImage(opts.Source, source)
	if err != nil {
		return "", fmt.Errorf("read local image %s: %w", source.Name(), err)
	}
	defer cleanup()
	localPort, stopForward, err := startRegistryPortForward(ctx, opts.Context, opts.Kubeconfig, opts.Stderr)
	if err != nil {
		return "", err
	}
	defer stopForward()

	pfRef, err := name.ParseReference("127.0.0.1:"+localPort+"/"+remoteRef.Context().RepositoryStr()+":"+remoteRef.Identifier(), name.Insecure)
	if err != nil {
		return "", fmt.Errorf("build port-forward registry reference: %w", err)
	}
	auth := authn.FromConfig(authn.AuthConfig{RegistryToken: token})
	if err := remote.Write(pfRef, img, remote.WithContext(ctx), remote.WithAuth(auth)); err != nil {
		return "", fmt.Errorf("push image: %w", err)
	}
	return remoteRef.Name(), nil
}
