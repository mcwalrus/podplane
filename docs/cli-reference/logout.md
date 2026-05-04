---
title: "logout"
weight: 31
description: "Remove cluster authentication from kubeconfig"
---

## Overview

Removes the previously authenticated cluster from your kubeconfig via kubectl.

```
podplane logout [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --cluster-config string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |
