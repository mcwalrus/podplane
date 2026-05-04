---
title: "cluster delete"
weight: 11
description: "Remove deployed cluster infrastructure and generated files"
---

## Overview

Removes deployed infrastructure (on AWS/Google Cloud) and deletes generated files created by `podplane cluster create`.

```
podplane cluster delete [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --cluster-config string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |
