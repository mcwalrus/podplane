// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"fmt"
	"regexp"
	"strings"
)

// dnsLabelPattern matches slash-free lowercase DNS-label-like path segments.
var dnsLabelPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// ValidateKey validates a Podplane secret key name.
func ValidateKey(key string) error {
	return validateDNSLabel("KEY", key, 63)
}

// ValidateProviderName validates a configured secrets provider name.
func ValidateProviderName(provider string) error {
	return validateDNSLabel("--provider", provider, 32)
}

// ValidateScopeName validates a namespace or SecretProviderClass scope name.
func ValidateScopeName(name, value string) error {
	return validateDNSLabel(name, value, 63)
}

// validateDNSLabel validates a lowercase DNS-label-like value with a maximum length.
func validateDNSLabel(name, value string, max int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(value) > max || strings.Contains(value, "--") || !dnsLabelPattern.MatchString(value) {
		return fmt.Errorf("%s must be lowercase alphanumeric with hyphens, no leading/trailing hyphens, no consecutive hyphens, and at most %d characters", name, max)
	}
	return nil
}
