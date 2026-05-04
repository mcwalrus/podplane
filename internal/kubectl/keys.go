// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package kubectl

import "fmt"

// ClusterKey returns the kubectl cluster key for the given clusterID.
// Remote: podplane-{clusterID}
// Local:  podplane-local-{clusterID}
func ClusterKey(clusterID string, local bool) string {
	if local {
		return fmt.Sprintf("podplane-local-%s", clusterID)
	}
	return fmt.Sprintf("podplane-%s", clusterID)
}

// ContextKey returns the kubectl context key for the given clusterID.
// Remote: {clusterID}
// Local:  local-{clusterID}, or just "local" when clusterID is "default"
func ContextKey(clusterID string, local bool) string {
	if local {
		if clusterID == "default" {
			return "local"
		}
		return fmt.Sprintf("local-%s", clusterID)
	}
	return clusterID
}

// CredentialsKey returns the kubectl credentials (user) key for the given
// sub and clusterID.
// Remote: podplane-{clusterID}-{sub}
// Local:  podplane-local-{clusterID}-{sub}
func CredentialsKey(sub string, clusterID string, local bool) string {
	if local {
		return fmt.Sprintf("podplane-local-%s-%s", clusterID, sub)
	}
	return fmt.Sprintf("podplane-%s-%s", clusterID, sub)
}
