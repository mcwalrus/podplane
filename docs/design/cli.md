---
title: "CLI Overview"
weight: 20
description: "Podplane CLI design and command overview"
---

# Podplane CLI

The Podplane CLI can be divided into groups of commands:

- `cluster` for managing Podplane clusters
- `oidc` for managing Easy OIDC deployments
- authentication commands
- `hooks` for integration e.g. kubectl exec auth plugin
- `local` for managing local VM clusters
- `package` for managing local VM dependency packages
- informational commands

Each CLI command group is summarised below.

## Config Files & Context

Many commands require either a cluster config file, or OIDC config file.

- For example, `podplane login` will require a cluster config file, and `podplane oidc delete` will require an oidc config file.

The CLI uses the current working directory to find the relevant `podplane.cluster.jsonc` or `podplane.oidc.jsonc` config file.

- Alternatively, you can specify a config file path using the `-f` flag e.g. `podplane login -f ./my-cluster/podplane.cluster.jsonc`

We recommend setting up a Git repository for storing all of your cluster and OIDC server infrastructure-as-code, for example:

```
├── infra/                          # git repo
│   │
│   ├── auth-production/            # example Easy OIDC server
│   │   ├── tf/                     # generated .tf files
│   │   └── podplane.oidc.jsonc     # config file
│   │
│   └── internaltools-production/   # example cluster
│       ├── tf/                     # generated .tf files
│       └── podplane.cluster.jsonc  # config file
```

## `cluster` commands

- `create` generates or reads a cluster config file, generates infra-as-code files, and (for AWS/Google Cloud) deploys the cluster via OpenTofu/Terraform
- `delete` for (on AWS/Google Cloud) removing deployed infrastructure and deleting generated files created by `create`.

## `oidc` commands

- `create` generates or reads an OIDC config file, generates infra-as-code files, and (for AWS/Google Cloud) deploys the OIDC via OpenTofu/Terraform
- `delete` for (on AWS/Google Cloud) removing deployed infrastructure and deleting generated files created by `create`.

## authentication commands

- `login` for authenticating via kubectl using the auth URL specified in the cluster configuration file
- `logout` to remove the previously authenticated cluster from your kubeconfig via kubectl

## `hooks` commands

- `kubectl-auth` to be used as a kubectl exec auth plugin

## `local` commands

You can run multiple single-node cluster VMs. If a name is omitted, `default` is used as the name.

- `start [name]` start a local cluster VM, and creates it if it doesn't exist
- `status [name]` report the status of a local cluster VM
- `stop [name]` stop a local cluster VM
- `delete [name]` delete a local cluster VM and its state files

The following commands exist primarily for Podplane development work on the `vmconfig` package:

- `shell [name]` open a shell into the local cluster VM or run a command via ssh
- `sync [name]` rsync files into the local cluster VM

## `package` commands

The `local` commands automatically download and cache packages. These commands exist primarily for debugging that cache.

- `status` reports current state of the cache and if any new package versions are available to download
- `download` force-downloads the latest package versions
- `server` runs a local webserver serving packages from cache

Note `server` is run automatically in the background when `local start` is used, and stopped on `local stop` of the last running VM.

## informational commands

- `version` reports the current CLI version
