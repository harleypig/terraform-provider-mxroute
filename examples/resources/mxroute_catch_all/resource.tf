resource "mxroute_domain" "example" {
  domain = "example.com"
}

resource "mxroute_catch_all" "example" {
  domain = mxroute_domain.example.domain

  # Deliver mail sent to unknown addresses to a specific mailbox. Use
  # "fail" to reject it or "blackhole" to silently discard it instead;
  # address is required only when type is "address".
  type    = "address"
  address = "postmaster@example.com"
}
