terraform {
  required_version = ">= 1.6.0"
  required_providers = {
    "aws" = {
      source = "hashicorp/aws"
      version = ">= 6.0"
    }
    "podplane" = {
      source = "podplane/podplane"
      version = ">= 1.0.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
  allowed_account_ids = ["123456789012"]
}

data "aws_caller_identity" "current" {
}

data "aws_region" "current" {
}

variable "mutable_env_overrides" {
  description = "Additional or overriding vmconfig mutable.env values."
  type = map(string)
  default = {}
}

locals {
  cluster_name = "Test Cluster"
  cluster_id = "test-cluster"
  name_prefix = "test-cluster"
  aws_account_id = data.aws_caller_identity.current.account_id
  aws_region = data.aws_region.current.region
  netsy_bucket_name = "${local.cluster_id}-${local.aws_account_id}-netsy"
  registry_bucket_name = "${local.cluster_id}-${local.aws_account_id}-registry"
  oidc_issuer_url = "https://auth.example.com"
  oidc_client_id = "test-cluster"
  oidc_username_claim = "email"
  oidc_groups_claim = "groups"
  kubernetes_api_hostname = "test-cluster.k8s.local"
  kubernetes_api_port = 6443
  kubernetes_cluster_cidr = []
  kubernetes_service_cidr = []
  mutable_env = merge({
    SSH_AUTHORIZED_KEY = ""
    KUBE_API_PUBLIC_HOSTNAME = local.kubernetes_api_hostname
    KUBE_API_PORT = tostring(local.kubernetes_api_port)
    KUBE_API_INTERNAL_LB_HOSTNAME = ""
    NSTANCE_SERVER_REGISTRATION_ADDR = ""
    NSTANCE_SERVER_AGENT_ADDR = ""
    KUBE_API_ETCD_SERVERS = ""
    OIDC_ISSUER = local.oidc_issuer_url
    OIDC_CUSTOM_CA = ""
    OIDC_CA_FILE = ""
    KUBE_LOG_LEVEL = "2"
  
    NETSY_BUCKET = aws_s3_bucket.netsy.bucket
    NETSY_ENDPOINT = ""
    NETSY_KEY_PREFIX = ""
    NETSY_ASSUME_ROLE = aws_iam_role.netsy.arn
    NETSY_REGION = local.aws_region
    NETSY_ACCESS_KEY_ID = ""
    NETSY_SECRET_ACCESS_KEY = ""
  
    TELEMETRY_ENABLED = "false"
    TELEMETRY_LOG_SERVICES = ""
    TELEMETRY_LOG_CLOUDINIT = "true"
    TELEMETRY_S3_BUCKET = ""
    TELEMETRY_S3_ENDPOINT = ""
    TELEMETRY_S3_REGION = local.aws_region
    TELEMETRY_S3_ASSUME_ROLE = ""
    TELEMETRY_S3_ACCESS_KEY_ID = ""
    TELEMETRY_S3_SECRET_ACCESS_KEY = ""
    TELEMETRY_OTLP_ENDPOINT = ""
  
    REGISTRY_ENABLED = "true"
    REGISTRY_BUCKET = aws_s3_bucket.registry.bucket
    REGISTRY_HOSTNAME = ""
    REGISTRY_ENDPOINT = ""
    REGISTRY_REGION = local.aws_region
    REGISTRY_ASSUME_ROLE = aws_iam_role.registry_read_only.arn
    REGISTRY_ACCESS_KEY_ID = ""
    REGISTRY_SECRET_ACCESS_KEY = ""
    AWS_S3_USE_PATH_STYLE = ""
  }, var.mutable_env_overrides)
}

module "cluster" {
  source = "nstance-dev/nstance/aws//modules/cluster"
  cluster_id = local.cluster_id
  name_prefix = local.name_prefix
}

output "cluster_id" {
  value = local.cluster_id
}

output "kubernetes_api_url" {
  value = "https://${local.kubernetes_api_hostname}:${local.kubernetes_api_port}"
}

module "account_123456789012_us_east_1" {
  source = "nstance-dev/nstance/aws//modules/account"
  cluster = module.cluster
}

resource "aws_s3_bucket" "netsy" {
  bucket = local.netsy_bucket_name
}

resource "aws_s3_bucket_public_access_block" "netsy" {
  bucket = aws_s3_bucket.netsy.id
  block_public_acls = true
  block_public_policy = true
  ignore_public_acls = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "netsy" {
  bucket = aws_s3_bucket.netsy.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket" "registry" {
  bucket = local.registry_bucket_name
}

resource "aws_s3_bucket_public_access_block" "registry" {
  bucket = aws_s3_bucket.registry.id
  block_public_acls = true
  block_public_policy = true
  ignore_public_acls = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "registry" {
  bucket = aws_s3_bucket.registry.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

data "aws_iam_policy_document" "assume_from_knc" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type = "AWS"
      identifiers = [module.account_123456789012_us_east_1.knc_iam_role_arn]
    }
  }
}

resource "aws_iam_role" "netsy" {
  name = "${local.name_prefix}-netsy"
  assume_role_policy = data.aws_iam_policy_document.assume_from_knc.json
}

resource "aws_iam_role_policy" "netsy" {
  name = "${local.name_prefix}-netsy-policy"
  role = aws_iam_role.netsy.id
  policy = data.aws_iam_policy_document.netsy.json
}

resource "aws_iam_role" "registry_read_only" {
  name = "${local.name_prefix}-registry-read-only"
  assume_role_policy = data.aws_iam_policy_document.assume_from_knc.json
}

resource "aws_iam_role_policy" "registry_read_only" {
  name = "${local.name_prefix}-registry-read-only-policy"
  role = aws_iam_role.registry_read_only.id
  policy = data.aws_iam_policy_document.registry_read_only.json
}

resource "aws_iam_role" "registry_read_write" {
  name = "${local.name_prefix}-registry-read-write"
  assume_role_policy = data.aws_iam_policy_document.assume_from_knc.json
}

resource "aws_iam_role_policy" "registry_read_write" {
  name = "${local.name_prefix}-registry-read-write-policy"
  role = aws_iam_role.registry_read_write.id
  policy = data.aws_iam_policy_document.registry_read_write.json
}

data "aws_iam_policy_document" "netsy" {
  statement {
    sid = "NetsyS3ObjectOperations"
    actions = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:GetObjectAttributes", "s3:AbortMultipartUpload", "s3:ListMultipartUploadParts"]
    resources = ["${aws_s3_bucket.netsy.arn}/*"]
  }

  statement {
    sid = "NetsyS3BucketOperations"
    actions = ["s3:ListBucket", "s3:ListBucketMultipartUploads"]
    resources = [aws_s3_bucket.netsy.arn]
  }
}

data "aws_iam_policy_document" "registry_read_only" {
  statement {
    actions = ["s3:ListBucket", "s3:GetBucketLocation", "s3:ListBucketMultipartUploads"]
    resources = [aws_s3_bucket.registry.arn]
  }

  statement {
    actions = ["s3:GetObject", "s3:ListMultipartUploadParts"]
    resources = ["${aws_s3_bucket.registry.arn}/*"]
  }
}

data "aws_iam_policy_document" "registry_read_write" {
  statement {
    actions = ["s3:ListBucket", "s3:GetBucketLocation", "s3:ListBucketMultipartUploads"]
    resources = [aws_s3_bucket.registry.arn]
  }

  statement {
    actions = ["s3:GetObject", "s3:ListMultipartUploadParts", "s3:PutObject", "s3:DeleteObject", "s3:AbortMultipartUpload"]
    resources = ["${aws_s3_bucket.registry.arn}/*"]
  }
}

module "network_123456789012_us_east_1" {
  source = "nstance-dev/nstance/aws//modules/network"
  cluster = module.cluster
  vpc_cidr_ipv4 = "172.18.0.0/16"
  enable_ipv6 = true
  subnets = {
    "control-plane" = {
      "us-east-1a" = [{ ipv4_cidr = "172.18.1.0/24", nat_subnet = "public" }]
    }
    "nstance" = {
      "us-east-1a" = [{ ipv4_cidr = "172.18.20.0/28", nat_subnet = "public" }]
    }
    "public" = {
      "us-east-1a" = [{ ipv4_cidr = "172.18.10.0/28", public = true, nat_gateway = true }]
    }
  }
  load_balancers = {
    "public-control-plane" = { ports = [6443], subnets = "public", public = true }
  }
}

module "shard_us_east_1a" {
  source = "nstance-dev/nstance/aws//modules/shard"
  cluster = module.cluster
  account = module.account_123456789012_us_east_1
  network = module.network_123456789012_us_east_1
  shard = "us-east-1a"
  zone = "us-east-1a"
  templates = {
    "control-plane" = {
      kind = "knc"
      arch = "arm64"
      args = { ImageId = "{{ .Image.debian_13_arm64 }}" }
    }
  }
  groups = {
    "default" = {
      "control-plane" = {
        template = "control-plane"
        size = 1
        subnet_pool = "control-plane"
        instance_type = "t4g.medium"
        vars = local.mutable_env
        load_balancers = ["public-control-plane"]
      }
    }
  }
}

resource "podplane_netsy_seed_s3" "cluster" {
  cluster_config_path = "${path.module}/podplane.cluster.jsonc"
  bucket = aws_s3_bucket.netsy.bucket
  region = "us-east-1"
  depends_on = [aws_s3_bucket.netsy]
}

output "nstance_bucket" {
  value = module.cluster.bucket
}

output "nstance_shards" {
  value = {
    "us-east-1a" = {
      config_key = module.shard_us_east_1a.config_key
      server_ips = module.shard_us_east_1a.server_ips
    }
  }
}

output "mutable_env" {
  value = local.mutable_env
}

output "registry_read_only_role_arn" {
  value = aws_iam_role.registry_read_only.arn
}

output "registry_read_write_role_arn" {
  value = aws_iam_role.registry_read_write.arn
}

output "netsy_role_arn" {
  value = aws_iam_role.netsy.arn
}
