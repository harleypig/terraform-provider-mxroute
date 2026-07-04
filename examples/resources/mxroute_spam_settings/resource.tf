resource "mxroute_domain" "example" {
  domain = "example.com"
}

resource "mxroute_spam_settings" "example" {
  domain = mxroute_domain.example.domain

  # Auto-delete mail scoring at or above this spam score (1-50).
  high_score = 15
}
