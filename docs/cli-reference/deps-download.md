---
title: "deps download"
weight: 81
description: "Download latest dependency versions"
---

## Overview

Force-downloads the latest dependency versions into the local cache.

Provider-specific dependencies and addon components are opt-in. By default, `deps download` downloads only provider-neutral dependencies and core components.

- Use `--providers aws`, `--providers aws,google`, or `--providers all` to include provider-specific entries.
- Use `--addons traefik,snapshot` or `--addons all` to include specific or all addon components.
- Pass `-f/--cluster-config <path>` specifying a cluster config file to infer providers from `cluster.providers[]`, addon components from `cluster.components.addons`, and ingress/certificate components needed by configured domains.

The local components images mirror only downloads images for the target architecture, such as `arm64` or `amd64`.

- You can specify one or both architectures with the `--arch` flag e.g. `--arch arm64,arm64` 
- For component images:
  - Some registry views may still show the full list of architectures from the original upstream image, but architectures you did not download are not actually available in the local mirror.
  - Use the mirror for the local VM architecture you downloaded, not as a complete copy of the upstream registry.

For development, pass `--vmconfig <path>` to use a local vmconfig manifest JSON file instead of fetching the vmconfig manifest from `cli.podplane.dev`. The referenced artifact URLs are still downloaded normally and the local vmconfig manifest is written into the deps cache on success. See [Development](../development.md) for more information.

```
podplane deps download [flags]
```
