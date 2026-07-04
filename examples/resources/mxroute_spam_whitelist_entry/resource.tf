resource "mxroute_spam_whitelist_entry" "example" {
  domain = "example.com"

  # A sender address or domain whose mail is always allowed.
  entry = "friend@good.example"
}
