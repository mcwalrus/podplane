---
title: "RBAC"
weight: 55
description: "Podplane Kubernetes access groups and default RBAC"
---

# Role-based Access Control (RBAC)

Podplane uses OIDC for Kubernetes authentication. The OIDC token issued to users includes a `groups` claim which Kubernetes uses to identify which group(s) the user is assigned to.

Kubernetes has multiple `ClusterRole` resources built-in, such as `cluster-admin`, `admin`, `edit`, and `view`. However, it does not define which external OIDC groups these roles map to.

Podplane provides default group bindings for these built-in roles. The `platform-rbac` component maps Podplane's default group names to the corresponding Kubernetes RBAC roles.

## Default Groups

| Group | Purpose | Kubernetes role |
| --- | --- | --- |
| `podplane:admins` | Unrestricted cluster administration. | `cluster-admin` |
| `podplane:managers` | Namespaced administration without full cluster-admin privileges. | `admin` |
| `podplane:operators` | Workload operations, including shell access to running pods. | `edit` |
| `podplane:editors` | Workload editing without shell access to running pods. | `edit` plus admission policy denying `pods/exec` and `pods/attach` |
| `podplane:viewers` | Read-only cluster access. | `view` |

`podplane:operators` and `podplane:editors` both map to Kubernetes `edit`. The distinction is enforced by `platform-rbac` admission policy: operators may use pod shell access, while editors may not unless they also belong to a higher shell-capable group.

## OIDC Requirements

Your OIDC issuer must let you assign users or teams to the group names above and emit those names in the token groups claim consumed by Kubernetes. [Easy OIDC](https://easy-oidc.dev) can be used for this, or you can bring an existing OIDC provider which supports custom groups claim configuration.

## Implementation

The `platform-rbac` [component](./components.md) installs the default `ClusterRoleBinding` resources and admission policies. Kubernetes remains the authorization source for API access; Podplane only standardises the group names and default bindings.
