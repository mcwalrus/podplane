---
title: "remove"
weight: 51
description: "Remove a previously deployed app"
---

## Overview

Removes a previously deployed app from the cluster.

This is a convenience command which wraps `helm` commands.

```
podplane remove <template> --name <name> [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `--name string` | Name of the app deployment to remove (required) |
| `--context string` | The name of the kubeconfig context to use |
| `--kubeconfig string` | Path to the kubeconfig file (default: `$KUBECONFIG` or `~/.kube/config`) |
