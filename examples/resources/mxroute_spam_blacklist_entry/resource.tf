resource "mxroute_spam_blacklist_entry" "example" {
  domain = "example.com"

  # A sender address or domain whose mail is always rejected.
  entry = "spammer@bad.example"
}
