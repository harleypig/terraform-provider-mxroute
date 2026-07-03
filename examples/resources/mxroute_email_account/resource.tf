resource "mxroute_domain" "example" {
  domain = "example.com"
}

resource "mxroute_email_account" "example" {
  domain   = mxroute_domain.example.domain
  username = "info"

  # password_wo is write-only: it is sent to the API but never stored in
  # Terraform state. Bump password_wo_version to rotate the password.
  password_wo         = "change-me-please"
  password_wo_version = 1

  quota = 2048 # megabytes
}
