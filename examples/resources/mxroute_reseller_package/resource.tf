resource "mxroute_reseller_package" "example" {
  name = "starter"

  # Limits are strings; use "unlimited" for no limit. Any omitted limit
  # inherits the API default for a new package.
  quota          = "5" # GB
  domains        = "10"
  email_accounts = "50"
}
