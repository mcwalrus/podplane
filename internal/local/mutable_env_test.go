// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"os"
	"testing"

	"github.com/podplane/podplane/internal/userdata"
)

func TestStageMutableEnvIfChangedStagesWhenBaselineDiffers(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	manager := &Local{dataDir: dataDir}
	store, err := newLocalNstanceStore(dataDir + "/nstance-fake")
	if err != nil {
		t.Fatalf("newLocalNstanceStore: %v", err)
	}

	desired := renderLocalMutableEnv(userdata.EnvVars{
		ClusterID:                     "cluster-a",
		OIDCIssuer:                    "https://10.0.2.2:1234/oidc",
		NstanceServerRegistrationAddr: "10.0.2.2:2345",
		NstanceServerAgentAddr:        "10.0.2.2:3456",
		NetsyEndpoint:                 "http://10.0.2.2:4567/s3/data",
		RegistryEndpoint:              "http://10.0.2.2:4567/s3/cache",
	})

	if err := manager.stageMutableEnvIfChanged(ctx, store, "cluster-a", "knc123", desired); err != nil {
		t.Fatalf("stageMutableEnvIfChanged: %v", err)
	}
	if _, err := os.Stat(manager.mutableEnvPath("cluster-a")); !os.IsNotExist(err) {
		t.Fatalf("baseline exists after staging: %v", err)
	}
	pending, err := getLocalPendingFiles(ctx, store, "knc123")
	if err != nil {
		t.Fatalf("getLocalPendingFiles: %v", err)
	}
	if len(pending) != 1 || pending[0].Filename != "mutable.env" || string(pending[0].Content) != desired {
		t.Fatalf("pending files = %#v, want mutable.env with desired content", pending)
	}
}

func TestWriteMutableEnvBaselineRecordsFirstBootWithoutStaging(t *testing.T) {
	dataDir := t.TempDir()
	manager := &Local{dataDir: dataDir}
	desired := "OIDC_ISSUER='https://10.0.2.2:1234/oidc'\n"

	if err := manager.writeMutableEnvBaseline("cluster-a", desired); err != nil {
		t.Fatalf("writeMutableEnvBaseline: %v", err)
	}
	if got, err := os.ReadFile(manager.mutableEnvPath("cluster-a")); err != nil || string(got) != desired {
		t.Fatalf("baseline = %q, %v; want desired content", got, err)
	}
}
