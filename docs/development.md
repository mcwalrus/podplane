---
title: "Development"
weight: 120
description: "Local development workflow for Podplane itself."
---

# Podplane Development

This guide is for folks working on Podplane itself: primarily when working on the [CLI](https://github.com/podplane/podplane), [vmconfig](https://github.com/podplane/vmconfig), and the [components](https://github.com/podplane/components), with local VMs used for development through the CLI `local` commands.

## Repository Layout

Use the [`podplane/workspace`](https://github.com/podplane/workspace) repository to keep the Podplane repositories checked out next to each other:

```text
workspace/
├── podplane/
│   ├── podplane/    # Podplane CLI
│   ├── vmconfig/    # VM install/configure scripts and dependency manifests
│   └── components/  # Helm charts and component manifests
├── netsy-dev/
...
```

The commands below assume you are in the `podplane` CLI repository and that the `../vmconfig` and `../components` repository checkouts exist.

## Local Cluster Development Flow

Before the first `local start`, download dependencies using the local `components` and `vmconfig` manifests:

```sh
go run . deps download \
  --components ../components/manifests/components.json \
  --vmconfig ../vmconfig/manifests/knc.debian-13.arm64.json
  # or for x86 dev machines:
  # --vmconfig ../vmconfig/manifests/knc.debian-13.amd64.json
```

By default this downloads provider-neutral dependencies and core component images only. If you are testing provider-specific components or addons locally, add filters such as `--providers aws` and `--addons traefik`.

Then start the local VM:

```sh
go run . local start
```

In a second terminal, run the vmconfig watch loop from the `vmconfig` repository:

```sh
cd ../vmconfig
make knc-watch
```

## Why This Works

`podplane deps download` normally fetches the published manifests from `https://cli.podplane.dev`. Passing local paths changes that behavior:

- `--components ../components/manifests/components.json` uses the local components manifest and mirrors the component images it names into the local dependency cache.
- `--vmconfig ../vmconfig/manifests/knc.debian-13.arm64.json` uses the local vmconfig manifest, which contains a vmconfig dependency stub instead of a released vmconfig tarball. That makes the local VM user-data skip extracting and running a prebuilt vmconfig package. For the OS image and other dependencies in the manifest, these will be mirrored into the local dependency cache, so using the local manifest means you can test out new dependencies as well as new configuration.

Note that the CLI checks the age of the cached manifests, so you need to re-run `deps download` on your development manifests at least once every 7 days to continue using them for new VMs.

With no prebuilt vmconfig package installed, the VM is ready for `make knc-watch`. That target watches the `vmconfig` templates, manifests, scripts, and Makefile; rebuilds the local `knc` tree on change; syncs it into the VM; and runs install/configure/restart as needed. Note that it will only run install once; to test that again, delete the VM with `go run . local stop --rm` and recreate it.

## What You Can Iterate On

This workflow gives you a local single-node control-plane VM that is useful for testing changes across Podplane's core repositories:

1. Iterate on the `components` manifest and test the images mirrored into the local dependency cache.
2. Iterate on the `vmconfig` manifest, templates, install scripts, and service configuration.
3. Test Podplane CLI changes for creating, starting, stopping, syncing, shelling into, and deleting local VMs.
4. Use the resulting bare Kubernetes cluster to test component bootstrap and addon installation.

Helpful commands from the CLI repository:

```sh
go run . local status
go run . local shell
go run . local console
go run . local stop
go run . local delete
# or perform stop+delete in a single command:
# go run . local stop --rm
```

Run `deps download` again before creating a fresh local VM whenever you change the local component or vmconfig manifests, or when you need to refresh mirrored component images.
