---
title: "login"
weight: 30
description: "Authenticate to a cluster via kubectl"
---

## Overview

Authenticates to a cluster via kubectl using the auth URL specified in the cluster configuration file. The authentication credentials are stored in your kubeconfig.

```
podplane login [flags]
```

## Options

| Flag | Description |
| --- | --- |
| `-f, --cluster-config string` | Path to the cluster config file (default: `podplane.cluster.jsonc` in the current directory) |
