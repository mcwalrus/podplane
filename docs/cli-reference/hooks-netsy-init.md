---
title: "hooks netsy-init"
weight: 41
description: "Generate an initial Netsy snapshot file from a template"
---

## Overview

Generates an initial Netsy snapshot file by downloading and interpolating a Netsy state template with cluster-specific settings from the cluster config file.

This command is typically not invoked directly. It is called by the Podplane Terraform/OpenTofu providers (one binary for AWS, one for Google Cloud) during cluster creation to produce the initial cluster state.

```
podplane hooks netsy-init [flags]
```

## How It Works

The Podplane providers for Terraform/OpenTofu are thin wrappers around this CLI command. The division of responsibility is:

1. **Podplane CLI** (`podplane hooks netsy-init`) handles downloading the Netsy state template file and interpolating it with cluster-specific settings - cluster name/slug, network configuration (IPv6 enabled, cluster CIDR, services CIDR), enabled components, etc. It outputs the interpolated snapshot file.

2. **Podplane TF Providers** handle uploading the snapshot to provider-specific object storage (S3 for AWS, GCS for Google Cloud). Before uploading, the provider checks that the first Netsy snapshot does not already exist, then performs a conditional put to create it. This ensures the provider can never overwrite a real snapshot by mistake.

Template interpolation lives in the CLI rather than the provider because the CLI also needs this functionality when creating local clusters via `podplane local start`.

## Options

| Flag | Description |
| --- | --- |
| `-f, --file string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |

## Related

- [Components - Cluster State Initialization](../components.md#cluster-state-initialization) for how initial state templates map to the Recommended, Minimal, and None options.
- [Infrastructure - Provisioning Flow](../infrastructure.md#provisioning-flow) for how this fits into the overall cluster creation flow.
