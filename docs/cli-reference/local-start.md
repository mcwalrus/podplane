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

```
podplane local start [flags]
```
