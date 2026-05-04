// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/podplane/podplane/internal/clusterconfig"
	"github.com/podplane/podplane/internal/deps"
	"github.com/podplane/podplane/internal/tui"
	"github.com/spf13/cobra"
)

var depsDownloadCmd = &cobra.Command{
	Use:          "download",
	Short:        "Downloads the latest dependency files",
	Long:         `Download fetches the latest dependency files. Generally you should not be running this directly.`,
	SilenceUsage: true,
}

var (
	depsDownloadDryRun             bool
	depsDownloadVMConfigManifest   string
	depsDownloadComponentsManifest string
	depsDownloadArchs              string
	depsDownloadProviders          string
	depsDownloadAddons             string
	depsDownloadClusterConfig      string
)

func init() {
	depsDownloadCmd.Flags().BoolVar(&depsDownloadDryRun, "dry-run", false,
		"Print what would be downloaded without actually downloading anything")
	depsDownloadCmd.Flags().StringVar(&depsDownloadVMConfigManifest, "vmconfig", "",
		"Path to a local vmconfig manifest JSON file to use instead of downloading the manifest")
	depsDownloadCmd.Flags().StringVar(&depsDownloadComponentsManifest, "components", "",
		"Path to a local components manifest JSON file to use instead of downloading the manifest")
	depsDownloadCmd.Flags().StringVar(&depsDownloadArchs, "arch", "",
		"Comma-separated target architectures to download (amd64,arm64). Defaults to the configured architecture")
	depsDownloadCmd.Flags().StringVar(&depsDownloadProviders, "providers", "",
		"Comma-separated provider-specific dependencies and component images to include (for example aws,google), or all")
	depsDownloadCmd.Flags().StringVar(&depsDownloadAddons, "addons", "",
		"Comma-separated addon component images to include (for example traefik,snapshot), or all")
	depsDownloadCmd.Flags().StringVarP(&depsDownloadClusterConfig, "cluster-config", "f", "",
		"Path to a cluster config file to infer providers and addon components")
}

func newDepsDownloadCmd(manager *deps.Manager, kind, arch string) *cobra.Command {
	depsDownloadCmd.RunE = func(cmd *cobra.Command, args []string) error {
		archs := []string{arch}
		if depsDownloadArchs != "" {
			archs = archs[:0]
			for _, part := range strings.Split(depsDownloadArchs, ",") {
				if arch := strings.TrimSpace(part); arch != "" {
					archs = append(archs, arch)
				}
			}
		}
		providers := []string{depsDownloadProviders}
		addons := []string{depsDownloadAddons}
		if depsDownloadClusterConfig != "" {
			cfg, err := clusterconfig.Load(depsDownloadClusterConfig)
			if err != nil {
				return err
			}
			for _, provider := range cfg.Cluster.Providers {
				providers = append(providers, provider.Kind)
			}
			for _, domain := range cfg.Cluster.Domains {
				providers = append(providers, domain.Provider.Kind)
			}
			if len(cfg.Cluster.Domains) > 0 {
				addons = append(addons, "cert-manager", "platform-certs", "traefik")
			}
			addons = append(addons, cfg.Cluster.Components.Addons...)
		}

		if depsDownloadDryRun {
			fmt.Println("Fetching latest dependency manifest...")
			for i, arch := range archs {
				err := manager.Download(kind, arch, deps.DownloadOptions{
					DryRun:                 true,
					Archs:                  archs,
					Providers:              providers,
					Addons:                 addons,
					SkipComponentImages:    i > 0,
					VMConfigManifestPath:   depsDownloadVMConfigManifest,
					ComponentsManifestPath: depsDownloadComponentsManifest,
				})
				if err != nil {
					fmt.Printf("Error downloading the latest dependency files: %v\n", err)
					return err
				}
			}
			return nil
		}

		err := tui.RunDownloadProgress("Downloading Podplane dependencies", func(progress func(deps.DownloadEvent)) error {
			for i, arch := range archs {
				err := manager.Download(kind, arch, deps.DownloadOptions{
					Archs:                  archs,
					Providers:              providers,
					Addons:                 addons,
					Progress:               progress,
					SkipComponentImages:    i > 0,
					VMConfigManifestPath:   depsDownloadVMConfigManifest,
					ComponentsManifestPath: depsDownloadComponentsManifest,
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error downloading the latest dependency files: %v\n", err)
		}
		return err
	}

	return depsDownloadCmd
}
