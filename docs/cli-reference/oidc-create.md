---
title: "oidc create"
weight: 20
description: "Generate OIDC configuration and deploy infrastructure"
---

## Overview

Generates or reads an OIDC config file, generates infrastructure-as-code files, and (for AWS/Google Cloud) deploys the OIDC server via OpenTofu/Terraform.

```
podplane oidc create [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --file string` | Path to the OIDC config file (default: `podplane.oidc.jsonc` in the current directory) |
