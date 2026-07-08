---
page_title: "Email Management"
description: |-
  Manage mailboxes, forwarders, catch-all policy, and spam filtering on an
  MXroute domain with the harleypig/mxroute provider's resources.
---

# Email Management

Manage the day-to-day mail objects on an MXroute domain — **mailboxes**,
**forwarders**, **catch-all policy**, and **spam filtering** — with the
`harleypig/mxroute` provider. It covers the same ground as MXroute's [Email
Accounts][mx-accounts], [Email Forwarders][mx-forwarders], and [Expert Spam
Filtering][mx-esf] guides, as Terraform.

New to the provider? Start with the [Quick Start](quick-start.md) — it
configures the provider (installed from the Terraform Registry) and stands up
a first domain. The examples below assume that provider configuration and a
managed domain (`mxroute_domain.primary`).

As always, this provider manages the **MXroute account side** through the API;
DNS records (MX / SPF / DKIM / DMARC) are set at your DNS provider — see the
Quick Start's [DNS records](quick-start.md#dns-records-set-these-at-your-dns-provider)
section.

## Mailboxes

A mailbox is an email account on a domain — a username, a password, and a
storage quota ([MXroute: Email Accounts][mx-accounts]). The
`mxroute_email_account` resource manages it.

The **password is write-only**: Terraform never records it in state, so it
must come from a sensitive variable, never a literal. `quota` is the storage
limit in MB (omit for unlimited); the optional `limit` caps daily sending. To
rotate the password, set the new value and bump `password_wo_version`.

```hcl
variable "alice_password" {
  type      = string
  sensitive = true
}

resource "mxroute_email_account" "alice" {
  domain   = mxroute_domain.primary.domain
  username = "alice"

  password_wo         = var.alice_password
  password_wo_version = 1

  quota = 5120 # MB; omit for unlimited
}
```

Changing a mailbox's `quota` (or rotating its password via
`password_wo_version`) is a normal `plan` / `apply` — the same
edit-then-apply loop the panel does by hand. MXroute enforces password
complexity at create (uppercase, lowercase, numbers, and a special
character).

## Forwarders

A forwarder (alias) redirects mail sent to one address to one or more
destinations, on the same or an external domain ([MXroute: Email
Forwarders][mx-forwarders]). The `mxroute_forwarder` resource manages it;
`destinations` is a set, so a single alias can fan out to several addresses.

```hcl
resource "mxroute_forwarder" "sales" {
  domain       = mxroute_domain.primary.domain
  alias        = "sales"
  destinations = ["alice@example.com", "team@example.net"]
}
```

MXroute's guidance applies: **test forwarders after applying**, and **avoid
forwarding loops** (chains that route back to themselves). A forwarder targets
a *specific* alias — distinct from a **catch-all**, which decides what happens
to mail sent to *undefined* addresses on the domain.

## Catch-all

The catch-all policy decides the fate of mail sent to any address on the
domain that isn't a mailbox or forwarder. The `mxroute_catch_all` resource
manages it, with three `type` values:

- `address` — deliver the mail to a specific mailbox (`address` is required).
- `fail` — reject the mail (the sender gets a bounce).
- `blackhole` — silently discard the mail.

```hcl
resource "mxroute_catch_all" "primary" {
  domain = mxroute_domain.primary.domain

  type    = "address"
  address = "alice@example.com"
}
```

Catch-all delivery can attract spam to the target mailbox; `fail` is the
conservative default when you don't need it.

## Spam filtering

MXroute has two distinct spam controls. One is managed by this provider; the
other is not — know which is which.

### SpamAssassin scoring and lists (provider-managed)

Per-domain SpamAssassin scoring plus black/whitelists, managed by three
resources:

- **`mxroute_spam_settings`** — `high_score`, the spam-score threshold
  (`1`–`50`); mail scoring at or above it is treated as spam, so a lower value
  filters more aggressively.
- **`mxroute_spam_blacklist_entry`** / **`mxroute_spam_whitelist_entry`** —
  per-domain `entry` patterns (an address or domain) to always reject or
  always allow.

```hcl
resource "mxroute_spam_settings" "primary" {
  domain     = mxroute_domain.primary.domain
  high_score = 5
}

resource "mxroute_spam_whitelist_entry" "newsletter" {
  domain = mxroute_domain.primary.domain
  entry  = "news@trusted.example"
}

resource "mxroute_spam_blacklist_entry" "spammer" {
  domain = mxroute_domain.primary.domain
  entry  = "spammer@bad.example"
}
```

The `mxroute_spam_blacklist` and `mxroute_spam_whitelist` data sources list a
domain's current entries.

### Expert Spam Filtering (not managed here)

**Expert Spam Filtering (ESF)** is a separate, **binary per-domain toggle**:
it rejects unauthenticated mail arriving from suspicious IP ranges (botnets,
compromised networks) at the SMTP level, and is enabled by default
([MXroute: Expert Spam Filtering][mx-esf]).

ESF is **not exposed by the `harleypig/mxroute` provider**, so **it does not
manage it**. Toggle it in the DirectAdmin panel (Spam Filters → Advanced), and
request whitelist exceptions for legitimately-blocked senders at
<https://esf.mxroute.com>. It is independent of the SpamAssassin scoring above
— changing `high_score` does not affect ESF, and vice versa.

## See also

- [Quick Start](quick-start.md) — provider setup and a first domain.
- Each resource's own page — `mxroute_email_account`, `mxroute_forwarder`,
  `mxroute_catch_all`, `mxroute_spam_settings`, and the rest — for the full
  field reference and import syntax.

[mx-accounts]: https://docs.mxroute.com/docs/email-accounts.html
[mx-forwarders]: https://docs.mxroute.com/docs/email-forwarders.html
[mx-esf]: https://docs.mxroute.com/docs/expert-spam-filtering.html
