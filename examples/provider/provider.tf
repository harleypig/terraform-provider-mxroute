provider "mxroute" {
  # All three may be omitted and read from the environment instead:
  #   MXROUTE_SERVER, MXROUTE_USERNAME, MXROUTE_API_KEY
  server   = "heracles.mxrouting.net"
  username = "myusername"
  api_key  = var.mxroute_api_key
}
