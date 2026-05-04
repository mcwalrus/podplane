// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package clusterconfig

import (
	"fmt"
	"regexp"
	"strings"
)

var clusterIDPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ValidateClusterID validates a cluster ID using Netsy's identifier rules.
func ValidateClusterID(id string) error {
	if id == "" {
		return fmt.Errorf("is required")
	}
	if len(id) > 32 {
		return fmt.Errorf("must be at most 32 characters")
	}
	if strings.Contains(id, "--") {
		return fmt.Errorf("must not contain consecutive hyphens")
	}
	if !clusterIDPattern.MatchString(id) {
		return fmt.Errorf("must be lowercase alphanumeric with hyphens, no leading/trailing hyphens")
	}
	if id == "local" || id == "k8s" || id == "oidc" {
		return fmt.Errorf("%q is reserved", id)
	}
	return nil
}

// ValidateComponents validates the optional components configuration.
func ValidateComponents(components Components) error {
	if components.Source == nil {
		return nil
	}
	refCount := 0
	if components.Source.Ref.Branch != "" {
		refCount++
	}
	if components.Source.Ref.Tag != "" {
		refCount++
	}
	if components.Source.Ref.Semver != "" {
		refCount++
	}
	if components.Source.Ref.Commit != "" {
		refCount++
	}
	if refCount > 1 {
		return fmt.Errorf("components.source.ref must set at most one of branch, tag, semver, or commit")
	}
	return nil
}
