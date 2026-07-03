resource "mxroute_forwarder" "example" {
  domain       = "example.com"
  alias        = "sales"
  destinations = ["owner@example.net", "team@example.net"]
}
