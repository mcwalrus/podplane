// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"fmt"
	"maps"
	"slices"
)

// EnableSet is the set of apps and CRDs that must be enabled to install a
// component. Names are sorted alphabetically.
type EnableSet struct {
	Apps []string
	CRDs []string
}

// IsEmpty reports whether the set is empty.
func (s EnableSet) IsEmpty() bool {
	return len(s.Apps) == 0 && len(s.CRDs) == 0
}

// ResolveEnable walks dependsOn transitively to compute the apps and CRDs
// that must be flipped to enabled=true so that name can be installed.
//
// The returned set excludes entries that are already enabled. If name itself
// is already enabled and all of its (transitive) dependencies are too, the
// returned set is empty.
//
// ResolveEnable returns an error when name is not present in the cluster's
// platform-components values, or when any (transitive) dependency cannot be
// found there.
func (c *Config) ResolveEnable(name string) (EnableSet, error) {
	if c == nil {
		return EnableSet{}, fmt.Errorf("components config is nil")
	}
	if !c.Has(name) {
		return EnableSet{}, fmt.Errorf("unknown component %q (not present in the platform-components HelmRelease values)", name)
	}

	apps := map[string]bool{}
	crds := map[string]bool{}
	visited := map[string]bool{}

	var walk func(string) error
	walk = func(n string) error {
		if visited[n] {
			return nil
		}
		visited[n] = true
		entry, isApp, ok := c.Get(n)
		if !ok {
			return fmt.Errorf("component %q has an unknown dependency", n)
		}
		if !entry.Enabled {
			if isApp {
				apps[n] = true
			} else {
				crds[n] = true
			}
		}
		for _, dep := range entry.DependsOn {
			if err := walk(dep); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(name); err != nil {
		return EnableSet{}, err
	}
	return EnableSet{Apps: slices.Sorted(maps.Keys(apps)), CRDs: slices.Sorted(maps.Keys(crds))}, nil
}

// EnabledDependents returns the names of currently enabled apps and CRDs
// whose dependsOn list contains name. Used by uninstall to refuse removing a
// component that other enabled components depend on.
func (c *Config) EnabledDependents(name string) []string {
	if c == nil {
		return nil
	}
	seen := map[string]bool{}
	visit := func(entries map[string]Entry) {
		for n, entry := range entries {
			if !entry.Enabled {
				continue
			}
			for _, dep := range entry.DependsOn {
				if dep == name {
					seen[n] = true
					break
				}
			}
		}
	}
	visit(c.Apps)
	visit(c.CRDs)
	return slices.Sorted(maps.Keys(seen))
}
