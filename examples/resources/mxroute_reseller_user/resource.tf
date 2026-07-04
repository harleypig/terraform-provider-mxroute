resource "mxroute_reseller_package" "example" {
  name = "starter"
}

resource "mxroute_reseller_user" "example" {
  username = "acustomer"
  email    = "owner@customer.example"
  package  = mxroute_reseller_package.example.name

  # password_wo is write-only: it is sent to the API but never stored in
  # Terraform state. Bump password_wo_version to rotate the password.
  password_wo         = "change-me-please"
  password_wo_version = 1

  quota = "500MB"
}
