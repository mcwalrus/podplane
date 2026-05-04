// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package qemu

import (
	"fmt"

	"github.com/podplane/podplane/internal/vm"
)

type Options struct {
	// Arch is the target architecture, e.g. "amd64" or "arm64".
	Arch string

	// ClusterID is used to derive a stable per-cluster VM name
	// ("podplane-local-<ClusterID>").
	ClusterID string

	// DataDir is the on-disk root for durable VM data (qcow2 images).
	// Concretely: <DataDir>/vms/<vmname>.<arch>.image.qcow2.
	DataDir string

	// RuntimeDir is the on-disk root for ephemeral VM runtime files (PIDs).
	// Concretely: <RuntimeDir>/vms/<vmname>.<arch>.pid.json.
	RuntimeDir string
}

// Qemu handles interactions with qemu
type Qemu struct {
	arch       string
	clusterID  string
	dataDir    string
	runtimeDir string
	VMName     string
}

// NewQemu creates a new qemu VM manager from explicit options.
func NewQemu(opts Options) vm.Manager {
	return &Qemu{
		arch:       opts.Arch,
		clusterID:  opts.ClusterID,
		dataDir:    opts.DataDir,
		runtimeDir: opts.RuntimeDir,
		VMName:     fmt.Sprintf("podplane-local-%s", opts.ClusterID),
	}
}
