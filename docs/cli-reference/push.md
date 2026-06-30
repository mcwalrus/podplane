---
title: "push"
---

# podplane push

Push a local image to the current cluster's container image registry through a Kubernetes port-forward.

```sh
podplane push <local-image> [<remote-image>]
```

If `<remote-image>` is omitted, Podplane pushes to:

```text
<cluster.registry.hostname>/apps/<source-repo-basename>:<source-tag-or-latest>
```

If `<remote-image>` omits a registry hostname, Podplane prefixes `cluster.registry.hostname`. Remote images must be under `apps/**`; `mirror/**` is reserved for Podplane-managed dependency mirrors.

The command requires the `zot-registry` component to be installed and ready in `platform-zot-registry` in the target cluster.
