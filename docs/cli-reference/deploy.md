---
title: "deploy"
weight: 50
description: "Deploy an app using a template"
---

## Overview

Deploys an app to the cluster using a template such as `web` or `worker`. The CLI will prompt to install addon components if the template has required dependencies which are not installed.

To update an existing app (e.g. to deploy a new image version), re-run this command with the same `--name`.

```
podplane deploy <template> --name <name> --image <image> [flags]
```

Under the hood this runs `helm upgrade --install`.

## Options

| Flag | Description |
| --- | --- |
| `--name string` | Name of the app deployment (required) |
| `--image string` | Container image to deploy (required) |
| `--context string` | The name of the kubeconfig context to use |
| `--kubeconfig string` | Path to the kubeconfig file (default: `$KUBECONFIG` or `~/.kube/config`) |
