output "oidc_issuer_url" {
  value = "https://${module.oidc.oidc_addr}"
}
