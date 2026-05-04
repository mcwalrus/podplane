// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/puidv7/puidv7-go"

	"github.com/podplane/podplane/internal/deps"
	"github.com/podplane/podplane/internal/osboot"
	"github.com/podplane/podplane/internal/userdata"
	"github.com/podplane/podplane/internal/vm"
)

// StartOptions controls local cluster startup.
type StartOptions struct {
	CPUs                string
	Memory              string
	StreamUserdataLogs  bool
	RunDownloadProgress func(run func(progress func(deps.DownloadEvent)) error) error
}

// Start is used to create a cluster, create a VM, and start a VM.
// Each cluster requires:
// 1.a. The package files to be downloaded to cache
// 1.b. The VM machine image to be downloaded to cache
// 2. CLI to be running a fake S3 and OIDC and package cache server in the background
// 3. The VM to be created
// 4. The VM to be started
// Start brings up the local cluster VM and writes a .cluster.jsonc config
// file describing how to log in to it. The returned path is the absolute
// location of that config (empty string on early failure paths). Callers
// (specifically `podplane local start`) use it to drive an in-process
// `podplane login --headless` against the local fake OIDC.
func (m *Local) Start(opts StartOptions) (string, error) {
	clusterID := m.clusterID
	if clusterID == "" {
		return "", fmt.Errorf("clusterID must be set")
	}

	// Verify cached deps. If anything is missing or corrupt, auto-run a
	// download so the user doesn't need to invoke `podplane deps download`
	// before their first `local start`. After that, `local start` is
	// offline-friendly: it never makes a network call for deps as part of
	// the main flow.
	depsManager := deps.NewManager(m.depsBaseURL, m.depsCacheDir)
	kind := m.instanceKind
	arch := m.arch

	manifest, err := depsManager.Verify(kind, arch)
	if errors.Is(err, deps.ErrNotCached) || errors.Is(err, deps.ErrIncomplete) {
		download := func(progress func(deps.DownloadEvent)) error {
			return depsManager.Download(kind, arch, deps.DownloadOptions{Progress: progress})
		}
		if opts.RunDownloadProgress != nil {
			err = opts.RunDownloadProgress(download)
		} else {
			err = download(nil)
		}
		if err != nil {
			return "", fmt.Errorf("failed to download deps: %w", err)
		}
		manifest, err = depsManager.Verify(kind, arch)
	}
	if err != nil {
		return "", fmt.Errorf("failed to verify deps: %w", err)
	}
	componentsEntries, componentsReadErr := os.ReadDir(depsManager.ComponentsImagesCacheDir())
	if componentsReadErr != nil || len(componentsEntries) == 0 {
		download := func(progress func(deps.DownloadEvent)) error {
			return depsManager.Download(kind, arch, deps.DownloadOptions{Progress: progress})
		}
		if opts.RunDownloadProgress != nil {
			err = opts.RunDownloadProgress(download)
		} else {
			err = download(nil)
		}
		if err != nil {
			return "", fmt.Errorf("failed to download component image deps: %w", err)
		}
	}

	// If the cached manifest is more than 7 days old, kick off a background
	// check for newer versions. The goroutine never blocks the main flow and
	// surfaces a non-blocking note at the end of Start. Skipped entirely when
	// the manifest is fresh, so we don't pay for goroutine setup or HTTP.
	nudgeCh := make(chan string, 1)
	close(nudgeCh) // default: no nudge
	if depsManager.IsStale(kind, arch, 7*24*time.Hour) {
		nudgeCh = make(chan string, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			nudgeCh <- depsManager.CheckUpdateNudge(ctx, kind, arch)
		}()
	}

	// Start the local server as a background process if not already running
	err = m.ServerEnsure()
	if err != nil {
		return "", fmt.Errorf("failed to start background server for local clusters: %w", err)
	}

	// Determine the host machine address from inside the guest machine.
	hostMachineAddr := m.vm.Addr()

	// Get URLs - note that all errors after the first are the same path.
	depsServerURL, err := m.DepsServerURL(hostMachineAddr, "")
	if err != nil {
		return "", fmt.Errorf("Unexpectedly failed to get URL of server for local clusters (maybe local server isn't running yet?): %w", err)
	}
	oidcIssuerURL, err := m.OIDCServerURL(hostMachineAddr)
	if err != nil {
		return "", fmt.Errorf("unexpectedly failed to get OIDC issuer URL for local clusters: %w", err)
	}
	s3DataEndpointURL, err := m.S3DataServerURL(hostMachineAddr)
	if err != nil {
		return "", fmt.Errorf("unexpectedly failed to get local data S3 endpoint URL: %w", err)
	}
	s3CacheEndpointURL, err := m.S3CacheServerURL(hostMachineAddr)
	if err != nil {
		return "", fmt.Errorf("unexpectedly failed to get local cache S3 endpoint URL: %w", err)
	}
	nstanceRegistrationAddr := replaceAddrHost(m.webserverPIDFile.GetData("nstance_registration_addr"), hostMachineAddr)
	nstanceAgentAddr := replaceAddrHost(m.webserverPIDFile.GetData("nstance_agent_addr"), hostMachineAddr)
	if nstanceRegistrationAddr == "" || nstanceAgentAddr == "" {
		return "", fmt.Errorf("local server is missing fake nstance address metadata; stop it with `podplane local server --stop` and retry")
	}

	// Check VM exists
	vmExisted, err := m.vm.Exists()
	if err != nil {
		return "", fmt.Errorf("failed to check if VM exists: %w", err)
	}
	if m.instanceID == "" && vmExisted {
		m.instanceID = m.existingInstanceID(clusterID)
	}
	if m.instanceID == "" {
		id, err := puidv7.New("knc")
		if err != nil {
			return "", fmt.Errorf("failed to generate instance ID: %w", err)
		}
		m.instanceID = id
	}
	instanceID := m.instanceID
	if !vmExisted {
		// Create the VM, using the cached OS image from the vmconfig manifest as
		// the qcow2 backing file.
		baseImage := depsManager.VMConfigArtifactCachePath(deps.ImageDepName, manifest.VMConfig.OS.Image)
		if err := m.vm.Create(baseImage); err != nil {
			_ = m.ServerCleanup()
			return "", fmt.Errorf("failed to create VM: %w", err)
		}
	}

	// Prefer direct kernel boot when the manifest provides explicit boot
	// metadata. If extraction fails, fall back to firmware/GRUB boot.
	var directBoot *vm.DirectBootOptions
	image := manifest.VMConfig.OS.Image
	boot := manifest.VMConfig.OS.Boot
	if boot.Complete() {
		imagePath := depsManager.VMConfigArtifactCachePath(deps.ImageDepName, image)
		directBoot, err = osboot.Prepare(osboot.Options{
			ImagePath: imagePath,
			CacheDir:  filepath.Join(filepath.Dir(imagePath), "boot"),
			Boot:      boot,
		})
		if err != nil {
			fmt.Printf("Direct boot unavailable, falling back to firmware boot: %v\n", err)
		}
	}

	// Get one key for ssh authorized_keys file
	sshAuthorizedKey, err := PubkeyForSshAuthorizedKey()
	if err != nil {
		sshAuthorizedKey = ""
	}

	// Read and base64-encode the local OIDC CA certificate.
	oidcCACertPath := m.OIDCCACertPath()
	certBytes, err := os.ReadFile(oidcCACertPath)
	if err != nil {
		return "", fmt.Errorf("failed to read local OIDC CA certificate file %s: %w", oidcCACertPath, err)
	}
	encodedCACert := base64.StdEncoding.EncodeToString(certBytes)

	// Configure the local fake Nstance deployment before rendering user-data. The
	// background `podplane local server` process owns the listening gRPC services;
	// this call opens the same durable store to write tenant config and bootstrap
	// state idempotently.
	nstanceBootstrap, err := configureLocalNstance(
		context.Background(),
		m.dataDir,
		clusterID,
		instanceID,
		kind,
		nstanceRegistrationAddr,
		nstanceAgentAddr,
		hostMachineAddr,
	)
	if err != nil {
		return "", fmt.Errorf("failed to configure local fake nstance: %w", err)
	}
	nstanceStore, err := newLocalNstanceStore(filepath.Join(m.dataDir, "nstance-fake"))
	if err != nil {
		return "", fmt.Errorf("failed to initialize local fake nstance store: %w", err)
	}

	// Render the user-data script.
	vars := userdata.TemplateVars{
		Manifest:                    manifest,
		DepsMirrorURL:               depsServerURL,
		NstanceRegistrationNonceJWT: nstanceBootstrap.RegistrationNonceJWT,
		Env: userdata.EnvVars{
			SSHAuthorizedKey:              sshAuthorizedKey,
			InstanceID:                    instanceID,
			ClusterID:                     clusterID,
			ProviderKind:                  "local",
			ProviderRegion:                "local",
			ProviderZone:                  "local",
			ProviderInstanceType:          "local",
			OIDCIssuer:                    oidcIssuerURL,
			OIDCCustomCA:                  encodedCACert,
			OIDCCAFile:                    "/opt/crt/oidc-ca.pem",
			KubeLogLevel:                  "5",
			KubeAPIPublicHostname:         "localhost",
			KubeAPIEtcdServers:            "https://127.0.0.1:2378",
			NstanceCACert:                 nstanceBootstrap.CACert,
			NstanceServerRegistrationAddr: nstanceBootstrap.ServerRegistrationAddr,
			NstanceServerAgentAddr:        nstanceBootstrap.ServerAgentAddr,
			TelemetryLogServices:          "first-boot-env,cron,ssh,netsy,nstance-agent,nstance-recv-watch,containerd,kube-apiserver,kube-controller-manager,kube-scheduler,kubelet,zot",
			RegistryEnabled:               "true",
			RegistryHostname:              fmt.Sprintf("%s-registry.local", clusterID),
			RegistryBucket:                "registry",
		},
	}
	vars.Env.SetObjectStorageEndpoint(s3DataEndpointURL)
	vars.Env.RegistryEndpoint = s3CacheEndpointURL
	vars.Env.SetObjectStorageRegion("local")
	vars.Env.SetObjectStorageCredentials("test", "test")
	vars.ApplyDefaults()
	mutableEnv := renderLocalMutableEnv(vars.Env)
	if vmExisted {
		if err := m.stageMutableEnvIfChanged(context.Background(), nstanceStore, clusterID, instanceID, mutableEnv); err != nil {
			return "", fmt.Errorf("failed to stage local mutable env update: %w", err)
		}
	}
	rendered, err := vars.Render()
	if err != nil {
		return "", fmt.Errorf("failed to render userdata: %w", err)
	}
	userdataFile := m.UserdataPath(clusterID)
	if err := os.MkdirAll(m.UserdataDir(clusterID), 0755); err != nil {
		return "", fmt.Errorf("failed to create userdata directory: %w", err)
	}
	if err := os.WriteFile(userdataFile, []byte(rendered), 0644); err != nil {
		return "", fmt.Errorf("failed to write user-data file %s: %w", userdataFile, err)
	}

	// Select host ports for the VM before starting it. Defaults are preferred for
	// single-VM local clusters, but occupied ports fall back to available dynamic
	// ports so multiple local VMs can run at the same time.
	vmPortForwards, vmPorts, err := allocateLocalVMPorts()
	if err != nil {
		return "", err
	}

	// Start the VM
	if err := m.vm.Start(rendered, opts.CPUs, opts.Memory, sshAuthorizedKey, false, directBoot, vmPortForwards); err != nil {
		if !errors.Is(err, vm.ErrAlreadyRunning) {
			_ = m.ServerCleanup()
		}
		return "", fmt.Errorf("failed to start VM: %w", err)
	}
	if !vmExisted {
		if err := m.writeMutableEnvBaseline(clusterID, mutableEnv); err != nil {
			return "", fmt.Errorf("failed to record local mutable env baseline: %w", err)
		}
	}
	if err := writeState(m.runtimeDir, clusterState{
		ClusterID: clusterID,
		Backend:   "qemu",
		Ports:     vmPorts,
	}); err != nil {
		return "", err
	}

	color.Green("✅ VM started successfully")
	if sshAuthorizedKey != "" {
		if err := m.WaitForReadiness(context.Background(), ReadinessOptions{
			StreamUserdataLogs: opts.StreamUserdataLogs,
		}); err != nil {
			return "", fmt.Errorf("local VM readiness check failed: %w", err)
		}
	} else {
		slog.Warn("skipping local VM readiness check because no SSH public key was available")
	}

	// Print any pending update nudge. Non-blocking: if the goroutine hasn't
	// produced a result yet (e.g. fast-fail before its 2s timeout), we drop
	// the message rather than delay return.
	select {
	case msg := <-nudgeCh:
		if msg != "" {
			fmt.Println(msg)
		}
	default:
	}

	// Write the .cluster.jsonc stash so the host-side `podplane login` (and
	// the in-process auto-login the cmd layer is about to run) can find this
	// local cluster by --cluster <id>. The OIDC issuer URL uses a shared
	// localhost hostname that is reachable from both the host CLI and the guest
	// VM's local-provider hosts entry.
	hostOIDCIssuer, err := m.HostOIDCIssuerURL()
	if err != nil {
		return "", fmt.Errorf("failed to derive host OIDC issuer URL: %w", err)
	}
	apiPort, err := strconv.Atoi(m.webserverPIDFile.GetData("ingress_https_port"))
	if err != nil {
		return "", fmt.Errorf("failed to derive local Kubernetes API ingress port: %w", err)
	}
	if apiPort == 0 {
		return "", fmt.Errorf("local server is missing ingress HTTPS port")
	}
	stashPath, err := m.WriteLocalClusterConfig(clusterID, hostOIDCIssuer, m.OIDCCACertPath(), LocalKubernetesAPIHostname(clusterID), apiPort)
	if err != nil {
		return "", fmt.Errorf("failed to write local cluster config: %w", err)
	}

	return stashPath, nil
}

var localUserdataInstanceIDPattern = regexp.MustCompile(`(?m)^INSTANCE_ID='([^']+)'$`)

// existingInstanceID returns the instance ID from the existing VM's user-data.
func (m *Local) existingInstanceID(clusterID string) string {
	data, err := os.ReadFile(m.UserdataPath(clusterID))
	if err != nil {
		return ""
	}
	match := localUserdataInstanceIDPattern.FindSubmatch(data)
	if len(match) != 2 {
		return ""
	}
	return string(match[1])
}

// WriteLocalClusterConfig writes a JSONC cluster config to
// <dataDir>/local/<clusterID>/cluster.jsonc and returns its absolute path. It
// describes how the host CLI can reach the local cluster's OIDC issuer and
// (eventually) Kubernetes API.
func (m *Local) WriteLocalClusterConfig(clusterID, oidcIssuerURL, oidcCACertPath, apiHostname string, apiPort int) (string, error) {
	dir := ClusterDataDir(m.dataDir, clusterID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	path := ClusterConfigPath(m.dataDir, clusterID)
	contents := fmt.Sprintf(`{
  // Auto-generated by `+"`"+`podplane local start`+"`"+` — describes the local
  // cluster so that `+"`"+`podplane login --cluster %s`+"`"+` (and the kubectl auth
  // hook) can find it.
  "cluster": {
    "id": %q,
    "name": %q,
    "oidc": {
      "issuer_url": %q,
      "client_id": %q,
      "username_claim": "email",
      "ca_cert": %q,
      "signing_algs": ["RS256"]
    },
    "domains": [
      {
        "zone": %q,
        "provider": { "kind": "local" }
      }
    ],
    "kubernetes": {
      "api_hostname": %q,
      "api_port": %d
    },
    "components": {
      "addons": []
    }
  }
}
`, clusterID, clusterID, "local-"+clusterID, oidcIssuerURL, clusterID, oidcCACertPath, clusterID+".localhost", apiHostname, apiPort)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

func replaceAddrHost(addr, host string) string {
	if addr == "" || host == "" {
		return addr
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return net.JoinHostPort(host, port)
}

// HostOIDCIssuerURL returns the OIDC issuer URL as reachable from the host
// machine (where the CLI itself runs), not from inside the guest VM.
func (m *Local) HostOIDCIssuerURL() (string, error) {
	port, err := m.LocalServerHTTPSPort()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s:%s/oidc", localOIDCHostname, port), nil
}
