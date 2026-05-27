// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"reflect"
	"testing"
)

// fixture builds a small Config used across resolve tests. The shape mirrors
// the platform-components values.yaml: traefik depends on platform-certs
// (addon) + traefik-crds + gateway-api-crds; platform-certs depends on
// cert-manager; cert-manager depends on cert-manager-crds.
func fixture() *Config {
	return &Config{
		Apps: map[string]Entry{
			"cilium":         {Enabled: true, Core: true, DependsOn: []string{"cilium-crds"}},
			"cert-manager":   {Enabled: false, DependsOn: []string{"cert-manager-crds"}},
			"platform-certs": {Enabled: false, DependsOn: []string{"cert-manager"}},
			"traefik":        {Enabled: false, DependsOn: []string{"platform-certs", "traefik-crds", "gateway-api-crds"}},
			"snapshot":       {Enabled: true, DependsOn: []string{"snapshot-crds"}},
		},
		CRDs: map[string]Entry{
			"cilium-crds":       {Enabled: true, Core: true},
			"cert-manager-crds": {Enabled: false},
			"traefik-crds":      {Enabled: false},
			"gateway-api-crds":  {Enabled: true, Core: true},
			"snapshot-crds":     {Enabled: true},
		},
	}
}

func TestResolveEnableTransitive(t *testing.T) {
	got, err := fixture().ResolveEnable("traefik")
	if err != nil {
		t.Fatal(err)
	}
	wantApps := []string{"cert-manager", "platform-certs", "traefik"}
	wantCRDs := []string{"cert-manager-crds", "traefik-crds"}
	if !reflect.DeepEqual(got.Apps, wantApps) {
		t.Errorf("apps = %v, want %v", got.Apps, wantApps)
	}
	if !reflect.DeepEqual(got.CRDs, wantCRDs) {
		t.Errorf("crds = %v, want %v", got.CRDs, wantCRDs)
	}
}

func TestResolveEnableAlreadyEnabled(t *testing.T) {
	got, err := fixture().ResolveEnable("snapshot")
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsEmpty() {
		t.Errorf("expected empty set, got apps=%v crds=%v", got.Apps, got.CRDs)
	}
}

func TestResolveEnableUnknown(t *testing.T) {
	if _, err := fixture().ResolveEnable("does-not-exist"); err == nil {
		t.Fatal("expected error for unknown component")
	}
}

func TestResolveEnableUnknownDependency(t *testing.T) {
	cfg := &Config{
		Apps: map[string]Entry{
			"broken": {Enabled: false, DependsOn: []string{"missing"}},
		},
	}
	if _, err := cfg.ResolveEnable("broken"); err == nil {
		t.Fatal("expected error for unknown transitive dep")
	}
}

func TestEnabledDependents(t *testing.T) {
	cfg := &Config{
		Apps: map[string]Entry{
			"cert-manager":   {Enabled: true, DependsOn: []string{"cert-manager-crds"}},
			"platform-certs": {Enabled: true, DependsOn: []string{"cert-manager"}},
			"traefik":        {Enabled: false, DependsOn: []string{"platform-certs"}},
		},
		CRDs: map[string]Entry{
			"cert-manager-crds": {Enabled: true},
		},
	}
	got := cfg.EnabledDependents("cert-manager")
	want := []string{"platform-certs"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dependents = %v, want %v", got, want)
	}
	if deps := cfg.EnabledDependents("platform-certs"); len(deps) != 0 {
		t.Errorf("platform-certs dependents = %v, want empty (traefik disabled)", deps)
	}
}
