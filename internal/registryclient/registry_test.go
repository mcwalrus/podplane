// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package registryclient

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

// TestStartRegistryPortForwardUsesKubectlAddressFlag verifies the port-forward
// command binds loopback with --address instead of embedding it in the mapping.
func TestStartRegistryPortForwardUsesKubectlAddressFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fake uses POSIX sh")
	}
	installFakeKubectlScript(t, `#!/bin/sh
found_address=false
found_service=false
found_mapping=false
for arg in "$@"; do
  if [ "$arg" = "--address" ]; then
    found_address=true
  fi
  if [ "$arg" = "svc/zot-registry" ]; then
    found_service=true
  fi
  case "$arg" in
    127.0.0.1:*:5000)
      printf 'address was embedded in port mapping: %s\n' "$arg" >&2
      exit 2
      ;;
    *:5000)
      found_mapping=true
      ;;
  esac
done
if [ "$found_address" != "true" ] || [ "$found_service" != "true" ] || [ "$found_mapping" != "true" ]; then
  printf 'unexpected args: %s\n' "$*" >&2
  exit 2
fi
printf 'Forwarding from 127.0.0.1:12345 -> 5000\n'
sleep 30
`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var output bytes.Buffer
	port, stop, err := startRegistryPortForward(ctx, "", "", &output)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if port == "" {
		t.Fatal("startRegistryPortForward() returned an empty port")
	}
	if !strings.Contains(output.String(), "Forwarding from") {
		t.Fatalf("forward output = %q, want readiness line", output.String())
	}
}

// TestWaitForPortForwardReadyReadsStdout verifies readiness detection handles
// kubectl versions that print port-forward status on stdout.
func TestWaitForPortForwardReadyReadsStdout(t *testing.T) {
	ready := make(chan error, 1)
	var output bytes.Buffer
	waitForPortForwardReady([]io.Reader{strings.NewReader("Forwarding from 127.0.0.1:12345 -> 5000\n"), strings.NewReader("")}, &output, ready)
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Forwarding from") {
		t.Fatalf("output = %q, want forwarded readiness line", output.String())
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
	installFakeKubectlScriptAt(t, path, script)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// installFakeKubectlScript installs a temporary kubectl shim backed by the
// supplied POSIX shell script.
func installFakeKubectlScript(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "kubectl")
	installFakeKubectlScriptAt(t, path, script)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// installFakeKubectlScriptAt writes a kubectl shim script at path.
func installFakeKubectlScriptAt(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
