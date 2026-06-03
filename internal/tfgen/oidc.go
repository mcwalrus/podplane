// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package tfgen

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/podplane/podplane/internal/oidcconfig"
)

// GenerateOIDC renders managed Terraform files for an OIDC config.
func GenerateOIDC(cfg *oidcconfig.Config) ([]File, error) {
	if err := oidcconfig.Validate(cfg); err != nil {
		return nil, err
	}
	if cfg.OIDC.Provider.Kind != "aws" {
		return nil, fmt.Errorf("OIDC provider %q is not supported", cfg.OIDC.Provider.Kind)
	}
	return []File{{Name: "podplane.oidc.tf", Content: renderAWSOIDC(cfg)}}, nil
}

// WriteOIDC writes managed Terraform files for an OIDC config.
func WriteOIDC(dir string, cfg *oidcconfig.Config) error {
	files, err := GenerateOIDC(cfg)
	if err != nil {
		return err
	}
	return WriteFiles(dir, files)
}

// renderAWSOIDC renders the AWS Easy OIDC Terraform file.
func renderAWSOIDC(cfg *oidcconfig.Config) string {
	o := cfg.OIDC
	var doc hclDocument

	terraform := block("terraform")
	terraform.Body.Attr("required_version", str(">= 1.6.0"))
	terraform.Body.Attr("required_providers", object(
		field("aws", object(
			identField("source", str("hashicorp/aws")),
			identField("version", str(">= 6.0")),
		)),
	))
	doc.AddBlock(terraform)

	provider := block("provider", "aws")
	provider.Body.Attr("region", str(o.Provider.Region))
	if o.Provider.Profile != "" {
		provider.Body.Attr("profile", str(o.Provider.Profile))
	}
	doc.AddBlock(provider)

	if o.Domain.Provider.Kind == "aws" && o.Domain.Zone != "" {
		zone := block("data", "aws_route53_zone", "oidc")
		if o.Domain.Provider.HostedZoneID != "" {
			zone.Body.Attr("zone_id", str(o.Domain.Provider.HostedZoneID))
		} else {
			zoneName := o.Domain.Zone
			if !strings.HasSuffix(zoneName, ".") {
				zoneName += "."
			}
			zone.Body.Attr("name", str(zoneName))
		}
		doc.AddBlock(zone)
	}

	module := block("module", "oidc")
	module.Body.Attr("source", str("easy-oidc/easy-oidc/aws"))
	module.Body.Attr("oidc_addr", str(hostOnly(o.Hostname)))
	module.Body.Attr("connector_type", str(o.Connector.Kind))
	module.Body.Attr("connector_client_secret_arn", str(o.Connector.ClientSecretARN))
	module.Body.Attr("signing_key_secret_arn", str(o.SigningKeySecretARN))
	if len(o.DefaultRedirectURIs) > 0 {
		module.Body.Attr("default_redirect_uris", stringValueList(o.DefaultRedirectURIs))
	}
	if len(o.Clients) > 0 {
		module.Body.Attr("clients", oidcClientsValue(o.Clients))
	}
	if len(o.GroupsOverrides) > 0 {
		module.Body.Attr("groups_overrides", groupsOverridesValue(o.GroupsOverrides))
	}
	if o.Domain.Provider.Kind == "aws" && o.Domain.Zone != "" {
		module.Body.Attr("route53_zone_id", expr("data.aws_route53_zone.oidc.zone_id"))
	}
	doc.AddBlock(module)

	output := block("output", "oidc_issuer_url")
	output.Body.Attr("value", str("https://${module.oidc.oidc_addr}"))
	doc.AddBlock(output)
	return doc.String()
}

// oidcClientsValue converts configured OIDC clients into an ordered HCL
// object.
func oidcClientsValue(clients map[string]oidcconfig.Client) hclObject {
	fields := make([]hclObjectField, 0, len(clients))
	for _, name := range sortedKeys(clients) {
		client := clients[name]
		clientFields := []hclObjectField{}
		if client.GroupsOverride != "" {
			clientFields = append(clientFields, identField("groups_override", str(client.GroupsOverride)))
		}
		if len(client.RedirectURIs) > 0 {
			clientFields = append(clientFields, identField("redirect_uris", stringValueList(client.RedirectURIs)))
		}
		fields = append(fields, field(name, object(clientFields...)))
	}
	return object(fields...)
}

// groupsOverridesValue converts group override config into an ordered HCL
// object.
func groupsOverridesValue(groups map[string]oidcconfig.GroupsOverride) hclObject {
	fields := make([]hclObjectField, 0, len(groups))
	for _, name := range sortedKeys(groups) {
		override := groups[name]
		overrideFields := make([]hclObjectField, 0, len(override))
		for _, email := range sortedKeys(override) {
			overrideFields = append(overrideFields, field(email, stringValueList(override[email])))
		}
		fields = append(fields, field(name, object(overrideFields...)))
	}
	return object(fields...)
}

// hostOnly returns the host component from a hostname or URL.
func hostOnly(hostname string) string {
	if !strings.Contains(hostname, "://") {
		return hostname
	}
	u, err := url.Parse(hostname)
	if err != nil || u.Host == "" {
		return hostname
	}
	return u.Host
}
