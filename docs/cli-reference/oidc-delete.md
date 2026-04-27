---
title: "oidc delete"
weight: 21
description: "Remove deployed OIDC infrastructure and generated files"
---

## Overview

Removes deployed infrastructure (on AWS/Google Cloud) and deletes generated files created by `podplane oidc create`.

```
podplane oidc delete [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --file string` | Path to the OIDC config file (default: `podplane.oidc.jsonc` in the current directory) |
