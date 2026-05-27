---
title: "local start"
weight: 70
description: "Start a local cluster VM"
---

## Overview

Starts a local single-node cluster VM, creating it if it doesn't exist. Packages are automatically downloaded and cached if not already present. A local server (serving packages and hosting a fake OIDC server) is started in the background.

Use `--console` to attach to the VM serial console after startup completes. This behaves like running [`podplane local console`](local-console.md) immediately after `local start`; press `Ctrl-]` to detach without stopping the VM.

Use `--follow` to stream `/var/log/cloud-init-output.log` while `local start` waits for cloud-init user-data to complete.

Use `--id` to select a non-default local cluster.

Use `--components` to select which platform components are seeded into Netsy on the very first boot. `recommended` (default) and `minimal` are seeded from the bundled `recommended.netsy` / `minimal.netsy` Podplane seed files. `none` skips seeding entirely; you get a bare cluster and are responsible for bootstrapping platform components yourself (for example, by running `DOMAIN=default.localhost make recommended` from the components repo against the default local cluster). For seeded clusters, the selected seed name and known available seed version are recorded in the auto-generated `cluster.jsonc` under `cluster.seed`; for `none`, `cluster.seed` is written as an empty object. Seeding is only performed on first boot — later runs of `podplane local start` never overwrite existing Netsy state.

```
podplane local start [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `--id string` | Local cluster ID (default: `default`) |
| `-c, --cpus string` | CPUs to allocate to the VM (default `2`) |
| `-m, --memory string` | Memory to allocate to the VM (default `4G`) |
| `--console` | Attach to the VM serial console after startup |
| `--follow` | Stream cloud-init user-data logs while waiting for startup |
| `--components string` | Platform components seeded on first boot: `recommended` (default), `minimal`, or `none`. |
