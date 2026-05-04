---
title: "local shell"
weight: 74
description: "Open a shell into a local cluster VM"
---

## Overview

Opens a shell into the local cluster VM or runs a command via SSH. This command exists primarily for Podplane development work on the `vmconfig` package.

This requires SSH access inside the guest. For boot debugging or before SSH is configured, use [`podplane local console`](local-console.md).

If a name is omitted, `default` is used.

```
podplane local shell [name] [flags]
```
