terraform {
  required_version = ">= 1.6.0"
  required_providers = {
    "aws" = {
      source = "hashicorp/aws"
      version = ">= 6.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

data "aws_route53_zone" "oidc" {
  name = "example.com."
}

module "oidc" {
  source = "easy-oidc/easy-oidc/aws"
  oidc_addr = "auth.example.com"
  connector_type = "google"
  connector_client_secret_arn = "arn:connector"
  signing_key_secret_arn = "arn:signing"
  default_redirect_uris = ["http://localhost:8000"]
  clients = {
    "kubelogin" = {}
  }
  route53_zone_id = data.aws_route53_zone.oidc.zone_id
}
