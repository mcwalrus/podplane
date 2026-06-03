// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"

	"github.com/podplane/podplane/internal/execwrap"
	"github.com/podplane/podplane/internal/vm/qemu"
)

// CheckRuntimeDependencies checks whether the local VM runtime dependencies
// are installed for arch.
func CheckRuntimeDependencies(arch string) error {
	return execwrap.Installed(qemu.QemuRequiredBinaries(arch))
}

// CheckServerRuntimeDependencies checks whether local server runtime
// dependencies are installed.
func CheckServerRuntimeDependencies() error {
	return execwrap.Installed([]string{"mkcert"})
}

// EnsureMkcertTrustInstalled installs mkcert's local CA into the host trust store.
// mkcert -install is idempotent and keeps local ingress certificates trusted by
// host browsers and HTTP clients.
func EnsureMkcertTrustInstalled() error {
	cmd := execwrap.Command("mkcert", "-install")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("install mkcert local CA: %w\n%s", err, string(output))
	}
	return nil
}
