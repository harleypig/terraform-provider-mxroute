terraform {
  # Write-only arguments (mxroute_email_account, mxroute_reseller_user)
  # require Terraform 1.11 or later.
  required_version = ">= 1.11"
}

provider "mxroute" {
  # All three may be omitted and read from the environment instead:
  #   MXROUTE_SERVER, MXROUTE_USERNAME, MXROUTE_API_KEY
  server   = "heracles.mxrouting.net"
  username = "myusername"
  api_key  = var.mxroute_api_key
}
