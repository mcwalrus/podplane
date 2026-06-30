// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package registryclient

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureZotRegistryReadySucceedsWithReadyEndpoint(t *testing.T) {
	installFakeKubectl(t, `{"subsets":[{"addresses":[{"ip":"10.0.0.10"}]}]}`)
	if err := ensureZotRegistryReady("test-context", "/tmp/kubeconfig"); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureZotRegistryReadyRejectsEmptyEndpoints(t *testing.T) {
	installFakeKubectl(t, `{"subsets":[{"addresses":[]}]}`)
	err := ensureZotRegistryReady("", "")
	if err == nil {
		t.Fatal("ensureZotRegistryReady() succeeded, want no ready endpoints error")
	}
	if !strings.Contains(err.Error(), "no ready Service endpoints") {
		t.Fatalf("error = %q, want no ready endpoints message", err)
	}
}

func installFakeKubectl(t *testing.T, stdout string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fake uses POSIX sh")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "kubectl")
	script := "#!/bin/sh\nprintf '%s' '" + strings.ReplaceAll(stdout, "'", "'\\''") + "'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
