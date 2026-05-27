---
title: "deps download"
weight: 81
description: "Download latest dependency versions"
---

## Overview

Force-downloads the latest dependency versions into the local cache.

Provider-specific dependencies are opt-in. By default, `deps download` downloads provider-neutral dependencies, core components, and the recommended addon component images needed by the default `recommended` seed.

- Use `--providers aws`, `--providers aws,google`, or `--providers all` to include provider-specific entries.
- Use `--addons snapshot` or `--addons all` to include extra addon components beyond the recommended defaults.
- Pass `-f/--cluster-config <path>` specifying a cluster config file to infer providers.

The local components images mirror only downloads images for the target architecture, such as `arm64` or `amd64`.

- You can specify one or both architectures with the `--arch` flag e.g. `--arch arm64,arm64` 
- For component images:
  - Some registry views may still show the full list of architectures from the original upstream image, but architectures you did not download are not actually available in the local mirror.
  - Use the mirror for the local VM architecture you downloaded, not as a complete copy of the upstream registry.

For development, pass `--vmconfig <path>`, `--components <path>`, or `--templates <path>` to use local manifest JSON files instead of fetching published manifests from `cli.podplane.dev`. See [Development](../development.md) for more information.

Pass `--skip-seeds` when to skip seed file downloads while still downloading VMConfig artifacts, component images, and template charts.

```
podplane deps download [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `--vmconfig string` | Path to a local vmconfig manifest JSON file |
| `--components string` | Path to a local components manifest JSON file |
| `--templates string` | Path to a local templates manifest JSON file |
| `--seeds string` | Path to a local seeds manifest JSON file |
| `--skip-seeds` | Skip downloading seed manifests and snapshots |
