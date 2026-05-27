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
| `-f, --oidc-config string` | Path to the OIDC config file (default: `podplane.oidc.jsonc` in the current directory) |
| `--no-apply` | Generate OpenTofu/Terraform files but do not run apply |
| `-y, --auto-approve` | Skip confirmation prompts and pass auto-approval to OpenTofu/Terraform |
