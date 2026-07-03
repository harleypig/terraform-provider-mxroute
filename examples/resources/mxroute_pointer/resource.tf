resource "mxroute_domain" "example" {
  domain = "example.com"
}

resource "mxroute_pointer" "example" {
  domain  = mxroute_domain.example.domain
  pointer = "www.example.com"
  alias   = true
}
