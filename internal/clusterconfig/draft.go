// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package clusterconfig

// NewDraftConfig returns a mutable draft cluster config for the requested
// provider.
func NewDraftConfig(providerKind string) *ClusterConfig {
	cfg := copyClusterConfig(defaultDraftConfig)
	if providerKind == "" {
		providerKind = "aws"
	}
	cfg.Cluster.Providers = []Provider{newDraftProvider(providerKind)}
	return &cfg
}

// newDraftProvider returns provider-specific draft infrastructure settings.
func newDraftProvider(kind string) Provider {
	switch kind {
	case "aws":
		return Provider{
			Kind:   "aws",
			Region: "us-east-1",
			VPC: VPC{
				V4CIDR: "172.18.0.0/16",
				V6CIDR: "auto",
			},
			Zones: map[string][]Subnet{
				"us-east-1a": {
					{V4CIDR: "172.18.10.0/28", V6CIDR: "auto", Services: []string{"nat", "nlb"}, Public: true},
					{V4CIDR: "172.18.20.0/28", V6CIDR: "auto", Services: []string{"nstance"}},
					{V4CIDR: "172.18.1.0/24", V6CIDR: "auto", Pool: "control-plane"},
				},
			},
			LoadBalancer: LoadBalancer{
				Public:    true,
				Listeners: []Listener{{Port: 6443, Pool: "control-plane"}},
			},
		}
	default:
		return Provider{Kind: kind}
	}
}

var defaultDraftConfig = ClusterConfig{Cluster: Cluster{
	ID:   "example-cluster",
	Name: "Example Cluster",
	OIDC: OIDC{IssuerURL: "https://auth.example.com"},
	Pools: map[string]Pool{
		"control-plane": {
			Arch:         "arm64",
			InstanceType: "t4g.medium",
			Size:         3,
		},
	},
	Kubernetes: Kubernetes{
		ClusterCIDR: []string{"100.64.0.0/10"},
		ServiceCIDR: []string{"198.18.0.0/15"},
	},
}}

// copyClusterConfig returns a deep enough copy for wizard mutation.
func copyClusterConfig(in ClusterConfig) ClusterConfig {
	out := in
	out.Cluster.Domains = append([]Domain(nil), in.Cluster.Domains...)
	out.Cluster.Providers = append([]Provider(nil), in.Cluster.Providers...)
	out.Cluster.Kubernetes.ClusterCIDR = append([]string(nil), in.Cluster.Kubernetes.ClusterCIDR...)
	out.Cluster.Kubernetes.ServiceCIDR = append([]string(nil), in.Cluster.Kubernetes.ServiceCIDR...)
	if in.Cluster.Components.Source != nil {
		source := *in.Cluster.Components.Source
		out.Cluster.Components.Source = &source
	}
	out.Cluster.Pools = make(map[string]Pool, len(in.Cluster.Pools))
	for name, pool := range in.Cluster.Pools {
		out.Cluster.Pools[name] = pool
	}
	return out
}
