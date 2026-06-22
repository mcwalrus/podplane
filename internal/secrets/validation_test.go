// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import "testing"

// TestValidateKey verifies Podplane secret key validation rules.
func TestValidateKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "simple", key: "database-url"},
		{name: "trimmed", key: " api-key "},
		{name: "empty", key: "", wantErr: true},
		{name: "uppercase", key: "API-key", wantErr: true},
		{name: "leading hyphen", key: "-api-key", wantErr: true},
		{name: "trailing hyphen", key: "api-key-", wantErr: true},
		{name: "consecutive hyphens", key: "api--key", wantErr: true},
		{name: "too long", key: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

// TestValidateProviderName verifies provider name validation and its shorter length limit.
func TestValidateProviderName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{name: "simple", provider: "aws-secrets-manager"},
		{name: "max length", provider: "abcdefghijklmnopqrstuvwxyzabcdef"},
		{name: "too long", provider: "abcdefghijklmnopqrstuvwxyzabcdefg", wantErr: true},
		{name: "invalid dns label", provider: "aws_secrets_manager", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderName(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateProviderName(%q) error = %v, wantErr %v", tt.provider, err, tt.wantErr)
			}
		})
	}
}

// TestValidateScopeName verifies namespace and binding scope name validation rules.
func TestValidateScopeName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "namespace", value: "default"},
		{name: "binding", value: "web-api"},
		{name: "dot rejected", value: "web.api", wantErr: true},
		{name: "slash rejected", value: "web/api", wantErr: true},
		{name: "too long", value: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScopeName("scope", tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateScopeName(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
