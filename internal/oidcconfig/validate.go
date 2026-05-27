// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package oidcconfig

import (
	"fmt"
	"net/url"
	"strings"
)

func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	o := cfg.OIDC
	switch o.Provider.Kind {
	case "aws":
		if o.Provider.Region == "" {
			return fmt.Errorf("oidc.provider.region is required for aws")
		}
	case "google", "gcp":
		// TODO: implement GCP support
		return fmt.Errorf("oidc.provider.kind google is not supported until the Easy OIDC Google module exists")
	case "":
		return fmt.Errorf("oidc.provider.kind is required")
	default:
		return fmt.Errorf("oidc.provider.kind %q is not supported", o.Provider.Kind)
	}
	if o.Hostname == "" {
		return fmt.Errorf("oidc.hostname is required")
	}
	if strings.Contains(o.Hostname, "://") {
		u, err := url.Parse(o.Hostname)
		if err != nil || u.Host == "" {
			return fmt.Errorf("oidc.hostname must be a hostname, not a URL")
		}
	}
	if o.Domain.Provider.Kind != "" && o.Domain.Provider.Kind != "aws" {
		return fmt.Errorf("oidc.domain.provider.kind %q is not supported for managed DNS yet", o.Domain.Provider.Kind)
	}
	switch o.Connector.Kind {
	case "google", "github":
	case "":
		return fmt.Errorf("oidc.connector.kind is required")
	default:
		return fmt.Errorf("oidc.connector.kind %q is not supported", o.Connector.Kind)
	}
	if o.Connector.ClientSecretARN == "" {
		return fmt.Errorf("oidc.connector.client_secret_arn is required")
	}
	if o.SigningKeySecretARN == "" {
		return fmt.Errorf("oidc.signing_key_secret_arn is required")
	}
	for name, client := range o.Clients {
		if name == "" {
			return fmt.Errorf("oidc.clients contains an empty client name")
		}
		if client.GroupsOverride != "" {
			if _, ok := o.GroupsOverrides[client.GroupsOverride]; !ok {
				return fmt.Errorf("oidc.clients.%s.groups_override references unknown groups override %q", name, client.GroupsOverride)
			}
		}
	}
	return nil
}
