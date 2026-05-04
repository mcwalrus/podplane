// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestLocalIngressClusterID(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		want       string
	}{
		{name: "cluster root", serverName: "dev.localhost", want: "dev"},
		{name: "host under cluster", serverName: "app.dev.localhost", want: "dev"},
		{name: "reserved kubernetes api", serverName: "dev.k8s.localhost", want: "dev"},
		{name: "case and trailing dot", serverName: "App.Dev.Localhost.", want: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := localIngressClusterID(tt.serverName)
			if err != nil {
				t.Fatalf("localIngressClusterID error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("localIngressClusterID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLocalIngressClusterIDRejectsInvalidHostnames(t *testing.T) {
	for _, serverName := range []string{"", "example.com", "localhost", "api.internal.dev.localhost"} {
		t.Run(serverName, func(t *testing.T) {
			if _, err := localIngressClusterID(serverName); err == nil {
				t.Fatalf("localIngressClusterID(%q) succeeded", serverName)
			}
		})
	}
}

func TestLocalIngressClusterIDAcceptsPort(t *testing.T) {
	got, err := localIngressClusterID("app.dev.localhost:4433")
	if err != nil {
		t.Fatalf("localIngressClusterID error = %v", err)
	}
	if got != "dev" {
		t.Fatalf("localIngressClusterID = %q, want dev", got)
	}
}

func TestLocalIngressTargetForKubernetesAPIHost(t *testing.T) {
	target, err := localIngressTargetForHost("dev.k8s.localhost:4433")
	if err != nil {
		t.Fatalf("localIngressTargetForHost error = %v", err)
	}
	if target.clusterID != "dev" || target.kind != localIngressTargetKubernetesAPI {
		t.Fatalf("target = %#v, want dev kubernetes api", target)
	}
}

func TestLocalIngressProxyRoutesKubernetesAPIHost(t *testing.T) {
	backend := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(backend.Close)

	port, err := strconv.Atoi(strings.TrimPrefix(backend.URL, "https://127.0.0.1:"))
	if err != nil {
		t.Fatalf("parse backend port from %q: %v", backend.URL, err)
	}
	runtimeDir := t.TempDir()
	if err := writeState(runtimeDir, clusterState{
		ClusterID: "dev",
		Backend:   "qemu",
		Ports:     portState{KubernetesAPI: port},
	}); err != nil {
		t.Fatalf("writeState: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "https://dev.k8s.localhost/readyz", nil)
	r.Host = "dev.k8s.localhost"
	w := httptest.NewRecorder()

	localIngressProxy(runtimeDir).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestLocalIngressProxyRoutesKubernetesAPIByTLSServerName(t *testing.T) {
	backend := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(backend.Close)

	port, err := strconv.Atoi(strings.TrimPrefix(backend.URL, "https://127.0.0.1:"))
	if err != nil {
		t.Fatalf("parse backend port from %q: %v", backend.URL, err)
	}
	runtimeDir := t.TempDir()
	if err := writeState(runtimeDir, clusterState{
		ClusterID: "dev",
		Backend:   "qemu",
		Ports:     portState{KubernetesAPI: port},
	}); err != nil {
		t.Fatalf("writeState: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "https://127.0.0.1:4433/readyz", nil)
	r.Host = "127.0.0.1:4433"
	r.TLS = &tls.ConnectionState{ServerName: "dev.k8s.localhost"}
	w := httptest.NewRecorder()

	localIngressProxy(runtimeDir).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestLocalIngressProxyShowsPlaceholderForMissingClusterState(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://hello.localhost:4433/", nil)
	w := httptest.NewRecorder()

	localIngressProxy(t.TempDir()).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	for _, want := range []string{"Podplane Local Ingress Proxy", `local cluster &#34;hello&#34; is not running`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("placeholder body missing %q: %s", want, w.Body.String())
		}
	}
}

func TestWriteLocalClusterConfigUsesReservedKubernetesAPIHost(t *testing.T) {
	manager := &Local{dataDir: t.TempDir()}
	path, err := manager.WriteLocalClusterConfig("dev", "https://oidc.localhost:1234/oidc", "/tmp/oidc-ca.pem", LocalKubernetesAPIHostname("dev"), 4433)
	if err != nil {
		t.Fatalf("WriteLocalClusterConfig: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cluster config: %v", err)
	}
	for _, want := range []string{`"issuer_url": "https://oidc.localhost:1234/oidc"`, `"ca_cert": "/tmp/oidc-ca.pem"`, `"api_hostname": "dev.k8s.localhost"`, `"api_port": 4433`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("cluster config missing %s:\n%s", want, string(data))
		}
	}
}

func TestLocalIngressPlaceholder(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "https://localhost:4433/", nil)
	w := httptest.NewRecorder()
	localIngressPlaceholder(w, r, errTestPlaceholder{})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if !strings.Contains(w.Body.String(), "Podplane Local Ingress Proxy") {
		t.Fatalf("placeholder body missing title: %s", w.Body.String())
	}
}

type errTestPlaceholder struct{}

func (errTestPlaceholder) Error() string { return "test reason" }
