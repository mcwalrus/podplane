// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	localIngressHTTPSPort      = 4433
	localTraefikHTTPSHostname  = "127.0.0.1"
	localKubernetesAPIHostname = "127.0.0.1"
)

type localIngressTargetKind string

const (
	localIngressTargetTraefik       localIngressTargetKind = "traefik"
	localIngressTargetKubernetesAPI localIngressTargetKind = "kubernetes-api"
)

type localIngressTarget struct {
	clusterID string
	kind      localIngressTargetKind
}

// LocalKubernetesAPIHostname returns the reserved host routed by the local
// ingress proxy to a cluster's Kubernetes API server.
func LocalKubernetesAPIHostname(clusterID string) string {
	return fmt.Sprintf("%s.k8s.localhost", clusterID)
}

// localIngressTargetForHost extracts the local cluster and target from an
// ingress hostname. App ingress uses <cluster-id>.localhost or
// <host>.<cluster-id>.localhost. The reserved <cluster-id>.k8s.localhost host
// routes to the Kubernetes API server and is intentionally outside the app
// ingress namespace.
func localIngressTargetForHost(host string) (localIngressTarget, error) {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return localIngressTarget{}, fmt.Errorf("local ingress TLS requires SNI")
	}
	if strings.HasSuffix(host, ".k8s.localhost") {
		clusterID := strings.TrimSuffix(host, ".k8s.localhost")
		if clusterID == "" || strings.Contains(clusterID, ".") {
			return localIngressTarget{}, fmt.Errorf("local Kubernetes API hostname %q must be <cluster-id>.k8s.localhost", host)
		}
		return localIngressTarget{clusterID: clusterID, kind: localIngressTargetKubernetesAPI}, nil
	}
	if !strings.HasSuffix(host, ".localhost") {
		return localIngressTarget{}, fmt.Errorf("local ingress TLS hostname %q is not under .localhost", host)
	}
	name := strings.TrimSuffix(host, ".localhost")
	parts := strings.Split(name, ".")
	if len(parts) > 2 {
		return localIngressTarget{}, fmt.Errorf("local ingress TLS hostname %q has too many labels before .localhost", host)
	}
	clusterID := parts[len(parts)-1]
	if clusterID == "" {
		return localIngressTarget{}, fmt.Errorf("local ingress TLS hostname %q does not include a cluster ID", host)
	}
	if clusterID == "k8s" {
		return localIngressTarget{}, fmt.Errorf("local ingress TLS hostname %q is reserved for Kubernetes API routing", host)
	}
	return localIngressTarget{clusterID: clusterID, kind: localIngressTargetTraefik}, nil
}

// localIngressClusterID extracts the local cluster ID from any valid local
// ingress hostname. It accepts either SNI-style hostnames or HTTP Host header
// values with ports.
func localIngressClusterID(host string) (string, error) {
	target, err := localIngressTargetForHost(host)
	if err != nil {
		return "", err
	}
	return target.clusterID, nil
}

// localIngressProxy builds the local TLS ingress reverse proxy to either the
// VM's raw Traefik HTTPS endpoint or the reserved Kubernetes API endpoint.
func localIngressProxy(runtimeDir string) http.Handler {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ingressTarget, err := localIngressTargetForHost(r.Host)
		if err != nil && r.TLS != nil && r.TLS.ServerName != "" {
			ingressTarget, err = localIngressTargetForHost(r.TLS.ServerName)
		}
		if err != nil {
			localIngressPlaceholder(rw, r, err)
			return
		}
		state, err := readState(runtimeDir, ingressTarget.clusterID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				localIngressPlaceholder(rw, r, fmt.Errorf("local cluster %q is not running", ingressTarget.clusterID))
				return
			}
			http.Error(rw, fmt.Sprintf("local ingress proxy failed to read cluster state: %v", err), http.StatusBadGateway)
			return
		}
		port, targetHost, targetName := state.Ports.TraefikHTTPS, localTraefikHTTPSHostname, "VM Traefik"
		if ingressTarget.kind == localIngressTargetKubernetesAPI {
			port, targetHost, targetName = state.Ports.KubernetesAPI, localKubernetesAPIHostname, "VM Kubernetes API"
		}
		if port == 0 {
			http.Error(rw, fmt.Sprintf("local ingress proxy failed to resolve %s port: state is missing port", targetName), http.StatusBadGateway)
			return
		}
		target := &url.URL{
			Scheme: "https",
			Host:   net.JoinHostPort(targetHost, strconv.Itoa(port)),
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = transport
		proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, err error) {
			message := fmt.Sprintf("local ingress proxy to %s is unavailable: %v", targetName, err)
			if ingressTarget.kind == localIngressTargetTraefik {
				message += "; ensure Traefik is installed and running in the local cluster"
			}
			http.Error(rw, message, http.StatusBadGateway)
		}
		proxy.ServeHTTP(rw, r)
	})
}

//go:embed ingress.html
var localIngressPlaceholderHTML string

// localIngressPlaceholder writes a static response for requests that reach the
// local ingress server without a valid local cluster ingress hostname.
func localIngressPlaceholder(rw http.ResponseWriter, _ *http.Request, reason error) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(rw, localIngressPlaceholderHTML, localIngressHTTPSPort, html.EscapeString(reason.Error()))
}
