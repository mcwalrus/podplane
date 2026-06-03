// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"testing"

	"github.com/podplane/podplane/internal/clusterconfig"
)

// TestParseOptions verifies Terraform output parsing for AWS cleanup options.
func TestParseOptions(t *testing.T) {
	outputs := []byte(`{
  "nstance_bucket": {
    "sensitive": false,
    "type": "string",
    "value": "cluster-bucket"
  },
  "nstance_shards": {
    "sensitive": false,
    "type": ["object", {}],
    "value": {
      "us-east-1a": {
        "config_key": "shard/us-east-1a/config.jsonc",
        "server_ips": ["10.0.0.10"]
      }
    }
  }
}`)
	cfg := &clusterconfig.ClusterConfig{Cluster: clusterconfig.Cluster{
		ID:        "test-cluster",
		Providers: []clusterconfig.Provider{{Kind: "aws", Region: "us-east-1", Profile: "default"}},
	}}
	opts, err := ParseOptions(cfg, outputs)
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if opts.Bucket != "cluster-bucket" {
		t.Fatalf("Bucket = %q, want cluster-bucket", opts.Bucket)
	}
	if opts.ShardConfigs["us-east-1a"].ConfigKey != "shard/us-east-1a/config.jsonc" {
		t.Fatalf("unexpected shard config: %#v", opts.ShardConfigs)
	}
}

// TestZeroGroups verifies that shard config group sizes are set to zero.
func TestZeroGroups(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(`{
  "groups": {
    "default": {
      "control-plane": { "size": 3 },
      "workers": { "size": 0 }
    }
  }
}`), &doc); err != nil {
		t.Fatal(err)
	}
	if !zeroGroups(doc) {
		t.Fatal("zeroGroups returned false, want true")
	}
	groups := doc["groups"].(map[string]any)["default"].(map[string]any)
	for name, value := range groups {
		group := value.(map[string]any)
		if group["size"] != float64(0) {
			t.Fatalf("%s size = %v, want 0", name, group["size"])
		}
	}
	if zeroGroups(doc) {
		t.Fatal("zeroGroups returned true after all sizes were already zero")
	}
}
