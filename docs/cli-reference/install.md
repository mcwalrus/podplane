---
title: "install"
weight: 60
description: "Install an addon component into the cluster"
---

## Overview

Installs an addon component into the cluster with an opinionated, tested configuration. Components are installed and managed by Flux CD.

```
podplane install <component> [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --file string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |
