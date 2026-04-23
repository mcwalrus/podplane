---
title: "Getting Started"
weight: 10
description: "Get started with Podplane quickly."
---

# Getting Started with Podplane

Podplane can deploy clusters on AWS, Google Cloud, or Proxmox environments.

Using the Podplane CLI, you can deploy a Podplane cluster in a few minutes, in one of two modes:

- __Kubernetes distribution__: minimal cluster so you can BYO stack.
    
    - Includes: Core DNS + Cilium CNI. BYO: Ingress controller, CSI drivers, secrets management, everything else!

- __Platform-as-a-Service (PaaS)__: a complete developer platform, ready to deploy your apps.
    
    - Includes: the base distribution, plus cert-manager, Traefik ingress, cloud-specific CSI drivers & snapshots controller, secrets store CSI driver, and more.

Deploying a cluster first generates versionable infrastructure-as-code artifacts such as OpenTofu/Terraform `.tf` files for AWS & Google Cloud, which then deploys a cluster into your public or private cloud of choice.

## Step 1: Install the Podplane CLI

macOS via [Homebrew](https://brew.sh/):

```bash
brew install podplane/podplane
```

or via [Go](https://go.dev/):

```bash
go install github.com/podplane/podplane@latest
```

## Step 2: Create Cluster

```bash
podplane create
```

Follow the prompts to specify:

- Which cloud/provider to use.
- Cloud provider config such as account/project/profile and region.
- Auth server URL, or opt to deploy a new [Easy OIDC](https://easy-oidc.dev) server.
- Cluster layout e.g. single node, separate control plane/ingress layers, etc.
- Networking configuration e.g. CIDR block for VPC and Subnet(s), provider zone(s).
- Default CPU architecture.
- Cluster name.
- Features e.g. Bare Kubernetes Distribution or PaaS
    - Some features may require additional info e.g. PaaS requires a default cluster domain

This will:

1. Create a `cluster.jsonc` file in the current directory
2. Generate the relevant infrastructure-as-code artifacts
3. For AWS/Google Cloud:
    1. Confirm if you want to immediately deploy
    2. Deploy using OpenTofu (or Terraform) `apply` command

Alternatively, `podplane create -f cluster.jsonc` can skip to step 2 for an existing cluster config file generated using `podplane init`.

## Step 3: Login

The Podplane CLI can automatically configure your local `kubeconfig` via `kubectl` using the login command:

```bash
podplane login
```

This will open a browser window (or print a URL to the terminal) to login via your configured OAuth provider. Once successful, you can then use all your favourite tools e.g.

```bash
kubectl get nodes --context cluster-name
```

## Step 4: Deploy Your App (PaaS)

If you selected PaaS for the cluster mode during creation, you can use the Podplane CLI to deploy your apps.

```bash
podplane deploy webapp --name test --image caddy
```

This will print a URL you can use to view the [Caddy server](https://caddyserver.com/) "Your web server is working" default page.

## Further Reading

Learn about how Podplane works in the [System Design](../design) docs.
