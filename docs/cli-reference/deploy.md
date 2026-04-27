---
title: "deploy"
weight: 50
description: "Deploy an app using a template"
---

## Overview

Deploys an app to the cluster using a template such as `web` or `worker`. The CLI will prompt to install addon components if the template has required dependencies which are not installed.

This is a convenience command which wraps `helm` commands.

```
podplane deploy <template> --name <name> --image <image> [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `--name string` | Name of the app deployment (required) |
| `--image string` | Container image to deploy (required) |
| `--context string` | The name of the kubeconfig context to use |
| `--kubeconfig string` | Path to the kubeconfig file (default: `$KUBECONFIG` or `~/.kube/config`) |
