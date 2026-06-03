// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"github.com/podplane/podplane/internal/pid"
	"github.com/podplane/podplane/internal/vm"
	"github.com/podplane/podplane/internal/vm/qemu"
)

// Local handles local cluster lifecycles
type Local struct {
	dataDir          string
	runtimeDir       string
	depsBaseURL      string
	depsCacheDir     string
	instanceKind     string
	arch             string
	clusterID        string
	instanceID       string
	vm               vm.Manager
	webserverPIDFile pid.PIDFile
}

// NewManager creates a new local cluster manager.
func NewManager(c ConfigSource, clusterID string) *Local {
	return &Local{
		dataDir:      c.DataDirectory(),
		runtimeDir:   c.RuntimeDirectory(),
		depsBaseURL:  c.DepsBaseURL(),
		depsCacheDir: c.DepsCacheDir(),
		instanceKind: c.InstanceKind(),
		arch:         c.Arch(),
		clusterID:    clusterID,
		vm: qemu.NewQemu(qemu.Options{
			Arch:       c.Arch(),
			ClusterID:  clusterID,
			DataDir:    c.DataDirectory(),
			RuntimeDir: c.RuntimeDirectory(),
		}),
	}
}
