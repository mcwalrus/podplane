// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nstance-dev/nstance/pkg/fakeserver"

	"github.com/podplane/podplane/internal/userdata"
)

const (
	localMutableEnvFilename = "mutable.env"
)

type localPendingFile struct {
	Filename     string    `json:"filename"`
	Content      []byte    `json:"content"`
	LastModified time.Time `json:"last_modified"`
}

// mutableEnvPath returns the per-cluster baseline mutable.env path. This file
// represents values that reached the VM through first boot or confirmed fake
// nstance delivery.
func (l *Local) mutableEnvPath(clusterID string) string {
	return filepath.Join(ClusterDataDir(l.dataDir, clusterID), localMutableEnvFilename)
}

// renderLocalMutableEnv renders the subset of user-data env that vmconfig's
// update-mutable-env.sh accepts for post-boot updates.
func renderLocalMutableEnv(env userdata.EnvVars) string {
	lines := []string{
		"SSH_AUTHORIZED_KEY=" + quoteLocalEnvValue(env.SSHAuthorizedKey),
		"KUBE_API_PUBLIC_HOSTNAME=" + quoteLocalEnvValue(env.KubeAPIPublicHostname),
		"KUBE_API_PORT=" + quoteLocalEnvValue(env.KubeAPIPort),
		"NSTANCE_SERVER_REGISTRATION_ADDR=" + quoteLocalEnvValue(env.NstanceServerRegistrationAddr),
		"NSTANCE_SERVER_AGENT_ADDR=" + quoteLocalEnvValue(env.NstanceServerAgentAddr),
		"KUBE_API_ETCD_SERVERS=" + quoteLocalEnvValue(env.KubeAPIEtcdServers),
		"OIDC_ISSUER=" + quoteLocalEnvValue(env.OIDCIssuer),
		"OIDC_CUSTOM_CA=" + quoteLocalEnvValue(env.OIDCCustomCA),
		"OIDC_CA_FILE=" + quoteLocalEnvValue(env.OIDCCAFile),
		"KUBE_LOG_LEVEL=" + quoteLocalEnvValue(env.KubeLogLevel),
		"NETSY_BUCKET=" + quoteLocalEnvValue(env.NetsyBucket),
		"NETSY_ENDPOINT=" + quoteLocalEnvValue(env.NetsyEndpoint),
		"NETSY_REGION=" + quoteLocalEnvValue(env.NetsyRegion),
		"NETSY_ASSUME_ROLE=" + quoteLocalEnvValue(""),
		"NETSY_ACCESS_KEY_ID=" + quoteLocalEnvValue(env.NetsyAccessKeyID),
		"NETSY_SECRET_ACCESS_KEY=" + quoteLocalEnvValue(env.NetsySecretAccessKey),
		"TELEMETRY_ENABLED=" + quoteLocalEnvValue("false"),
		"TELEMETRY_LOG_SERVICES=" + quoteLocalEnvValue(env.TelemetryLogServices),
		"TELEMETRY_LOG_CLOUDINIT=" + quoteLocalEnvValue(env.TelemetryLogCloudinit),
		"TELEMETRY_S3_BUCKET=" + quoteLocalEnvValue(env.TelemetryBucket),
		"TELEMETRY_S3_ENDPOINT=" + quoteLocalEnvValue(env.TelemetryEndpoint),
		"TELEMETRY_S3_REGION=" + quoteLocalEnvValue(env.TelemetryRegion),
		"TELEMETRY_S3_ASSUME_ROLE=" + quoteLocalEnvValue(""),
		"TELEMETRY_S3_ACCESS_KEY_ID=" + quoteLocalEnvValue(env.TelemetryAccessKeyID),
		"TELEMETRY_S3_SECRET_ACCESS_KEY=" + quoteLocalEnvValue(env.TelemetrySecretAccessKey),
		"TELEMETRY_OTLP_ENDPOINT=" + quoteLocalEnvValue(""),
		"REGISTRY_ENABLED=" + quoteLocalEnvValue(env.RegistryEnabled),
		"REGISTRY_BUCKET=" + quoteLocalEnvValue(env.RegistryBucket),
		"REGISTRY_HOSTNAME=" + quoteLocalEnvValue(env.RegistryHostname),
		"REGISTRY_ENDPOINT=" + quoteLocalEnvValue(env.RegistryEndpoint),
		"REGISTRY_REGION=" + quoteLocalEnvValue(env.RegistryRegion),
		"REGISTRY_ASSUME_ROLE=" + quoteLocalEnvValue(""),
		"REGISTRY_ACCESS_KEY_ID=" + quoteLocalEnvValue(env.RegistryAccessKeyID),
		"REGISTRY_SECRET_ACCESS_KEY=" + quoteLocalEnvValue(env.RegistrySecretAccessKey),
		"AWS_S3_USE_PATH_STYLE=" + quoteLocalEnvValue(env.AWSS3UsePathStyle),
	}
	return strings.Join(lines, "\n") + "\n"
}

// quoteLocalEnvValue quotes a value using the same single-quote format as the
// vmconfig shell helpers that write env files inside the VM.
func quoteLocalEnvValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// writeMutableEnvBaseline records content as already available to the VM.
func (l *Local) writeMutableEnvBaseline(clusterID, content string) error {
	path := l.mutableEnvPath(clusterID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

// stageMutableEnvIfChanged stages desired mutable.env through fake Nstance
// only when it differs from the delivered baseline for this cluster. The
// returned bool is true when desired was staged for eventual delivery.
func (l *Local) stageMutableEnvIfChanged(ctx context.Context, store fakeserver.Store, clusterID, instanceID, desired string) (bool, error) {
	delivered, err := os.ReadFile(l.mutableEnvPath(clusterID))
	if err == nil && string(delivered) == desired {
		return false, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := l.stageMutableEnv(ctx, store, clusterID, instanceID, desired); err != nil {
		return false, err
	}
	return true, nil
}

// stageMutableEnv inserts or replaces mutable.env in the fake Nstance pending
// file queue for eventual delivery to the agent.
func (l *Local) stageMutableEnv(ctx context.Context, store fakeserver.Store, clusterID, instanceID, content string) error {
	pendingFiles, err := getLocalPendingFiles(ctx, store, instanceID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	staged := false
	for i := range pendingFiles {
		if pendingFiles[i].Filename == localMutableEnvFilename {
			pendingFiles[i].Content = []byte(content)
			pendingFiles[i].LastModified = now
			staged = true
			break
		}
	}
	if !staged {
		pendingFiles = append(pendingFiles, localPendingFile{
			Filename:     localMutableEnvFilename,
			Content:      []byte(content),
			LastModified: now,
		})
	}
	data, err := json.Marshal(pendingFiles)
	if err != nil {
		return err
	}
	if err := store.Put(ctx, localPendingFilesKey(instanceID), data); err != nil {
		return err
	}
	return nil
}

// getLocalPendingFiles reads fake Nstance's pending-files JSON for an instance.
func getLocalPendingFiles(ctx context.Context, store fakeserver.Store, instanceID string) ([]localPendingFile, error) {
	data, err := store.Get(ctx, localPendingFilesKey(instanceID))
	if errors.Is(err, fakeserver.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pendingFiles []localPendingFile
	if err := json.Unmarshal(data, &pendingFiles); err != nil {
		return nil, fmt.Errorf("decode fake nstance pending files: %w", err)
	}
	return pendingFiles, nil
}

// localPendingFilesKey returns the fake Nstance store key for an instance's
// pending file queue.
func localPendingFilesKey(instanceID string) string {
	return filepath.ToSlash(filepath.Join("fakeserver", "instances", instanceID, "pending-files.json"))
}
