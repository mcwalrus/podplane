---
title: "hooks netsy-init"
weight: 41
description: "Generate an initial Netsy snapshot file from a template"
---

## Overview

Generates an initial Netsy snapshot file by downloading or reading a Netsy snapshot template and interpolating it with cluster-specific settings from the cluster config file.

This command is typically not invoked directly. It is called by the Podplane Terraform/OpenTofu providers during cluster creation to produce the initial Kubernetes state stored in Netsy.

```
podplane hooks netsy-init [flags]
```

## How It Works

Netsy implements a subset of the etcd API such that it is a compatible replacement for use with Kubernetes for storing cluster state. Netsy durably stores key/value records in `.netsy` data files in object storage. A chunk file is one or more records buffered, a snapshot file is a complete set of records up to a revision. For clusters with pre-existing Netsy state, when a control plane VM starts, Netsy restores the latest snapshot + any newer chunk files, before the kube-apiserver begins serving.

The Podplane providers for Terraform/OpenTofu are thin wrappers around this CLI command. The division of responsibility is:

1. **Podplane CLI** (`podplane hooks netsy-init`) downloads or reads a Netsy snapshot template, finds the `platform-components` Flux `HelmRelease` stored in that snapshot, interpolates the `spec.values.platform.components` values derived from the cluster config passed via `--cluster-config` (defaulting to `podplane.cluster.jsonc` in the current directory), and updates the bootstrap `podplane-components` `GitRepository` if a component source override is configured. For local clusters, `podplane local start` writes and passes an auto-generated config under Podplane's data directory.

2. **Podplane TF Providers** upload the resulting `.netsy` snapshot to provider-specific object storage (S3 for AWS, GCS for Google Cloud). Before uploading, the provider checks that the first Netsy snapshot does not already exist, then performs a conditional put to create it. This ensures the provider can never overwrite real cluster state by mistake.

Template interpolation lives in the CLI rather than the provider because the CLI also needs this functionality when creating local clusters via `podplane local start`.

For production domains with `cluster.acme`, the interpolated platform values configure cert-manager ACME DNS-01 solvers for AWS Route53, Cloudflare, or Google CloudDNS. For local domains without `cluster.acme`, they use the platform self-signed issuer; host-facing local TLS is handled separately by `podplane local server`.

## Example

```bash
podplane hooks netsy-init -f podplane.cluster.jsonc > 0000000000000000001.netsy
```

Or write directly to a file:

```bash
podplane hooks netsy-init -f podplane.cluster.jsonc -o 0000000000000000001.netsy
```

Merge additional platform-components Helm values over values derived from the cluster config:

```bash
podplane hooks netsy-init -f podplane.cluster.jsonc --values platform-values.yaml -o 0000000000000000001.netsy
```

Note: to read the output and verify the integrity of a `.netsy` file, there is a [read-netsy-file](https://github.com/netsy-dev/netsy/blob/main/cmd/read-netsy-file/main.go) utility available.

To use an explicit template, pass either a local path or an HTTP(S) URL:

```bash
podplane hooks netsy-init -f podplane.cluster.jsonc --template ./my-snapshot.netsy -o 0000000000000000001.netsy
podplane hooks netsy-init -f podplane.cluster.jsonc --template https://example.com/my-snapshot.netsy -o 0000000000000000001.netsy
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --cluster-config string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |
| `-o, --output string` | Path to write the generated Netsy snapshot file. Defaults to stdout. Use `-` for stdout. |
| `--template string` | Path or HTTP(S) URL for the Netsy snapshot template. Defaults to the published recommended template under the configured Podplane deps base URL. |
| `--values string` | YAML/JSON values file to merge over derived platform-components Helm values. |

## Related

- [Components - Cluster State Initialization](../components.md#cluster-state-initialization) for how initial state templates are used.
- [Infrastructure - Provisioning Flow](../infrastructure.md#provisioning-flow) for how this fits into the overall cluster creation flow.
