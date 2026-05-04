---
title: "cluster create"
weight: 10
description: "Generate cluster configuration and deploy infrastructure"
---

## Overview

Generates or reads a cluster config file, generates infrastructure-as-code files, and (for AWS/Google Cloud) deploys the cluster via OpenTofu/Terraform.

```
podplane cluster create [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --cluster-config string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |
