## 0.4.0

Compatibility: targets MXroute API 1.x.

BUG FIXES:

* resource/mxroute_domain, data-source/mxroute_domain: reading a domain that
  has one or more pointers no longer fails with `cannot unmarshal object into
  Go struct field Domain.pointers of type []string`. The live API returns a
  domain's `pointers` as an object keyed by pointer name (not the documented
  array of strings); both shapes now decode to the list of pointer names.

NOTES:

* docs: added two provider guides — **Quick Start** (stand up a domain,
  mailbox, and forwarder) and **Email Management** (mailboxes, forwarders,
  catch-all, and spam filtering) — rendered to the Registry from
  `templates/guides/`, using the provider's resources directly.
* resource/mxroute_email_account: the `password_wo` documentation now notes
  that the API enforces password complexity (a mix of uppercase, lowercase,
  numbers, and special characters) at create.
* resource/mxroute_domain, data-source/mxroute_domain: the `ssl_enabled`
  documentation is corrected — the MXroute API has no operation to request or
  issue a certificate, so the attribute is read-only status (certificates are
  provisioned out-of-band). Drops the previous unverifiable AutoSSL / ~24-hour
  timing claims.

## 0.3.0

Compatibility: targets MXroute API 1.x.

BREAKING CHANGES:

* resource/mxroute_forwarder: `destinations` is now a **set** rather than a
  list, so the API returning the addresses in a different order than configured
  no longer forces a spurious destroy/recreate of a live forwarder.

FEATURES:

* **New Data Source:** `mxroute_forwarders`
* **New Data Source:** `mxroute_pointers`
* **New Data Source:** `mxroute_spam_blacklist`
* **New Data Source:** `mxroute_spam_whitelist`
* **New Data Source:** `mxroute_reseller_package`
* **New Data Source:** `mxroute_reseller_user`

ENHANCEMENTS:

* resource/mxroute_email_account: `limit` is now sent on create (previously a
  `limit` set at create was silently dropped); `limit` gains a plan-time upper
  bound (9600) and `password_wo` a minimum length (8).
* resource/mxroute_reseller_user: `password_wo` is now optional — required only
  when creating a user — mirroring `mxroute_email_account`, with a create-time
  presence guard and a rotation guard, plus a minimum length (8).
* provider: rate-limited (429) responses with no `Retry-After` hint now back
  off exponentially instead of using a flat delay.

BUG FIXES:

* provider: user-controlled URL path segments (spam entries, forwarder aliases,
  mailbox and reseller names) are now percent-encoded, so a value containing
  `/`, `#`, `?`, or a space no longer breaks the request — most notably a delete
  that silently missed and left the resource in place.
* resource/mxroute_catch_all: an empty-string `address` is now treated as unset
  — `type = "address"` with `address = ""` is rejected, and `type =
  "fail"`/`"blackhole"` with `""` no longer errors.

NOTES:

* Internal DRY refactor — shared Configure / fetch / schema / ImportState
  helpers across the resources and data sources; no user-facing behavior change.

## 0.2.0

Compatibility: targets MXroute API 1.x.

ENHANCEMENTS:

* resource/mxroute_email_account: `password_wo` is now optional, so an existing
  mailbox no longer needs a password in its configuration. It is still required
  when **creating** a mailbox (enforced with a clear error, matching the API,
  which requires a password on create but not on update), and bumping
  `password_wo_version` without supplying a password is now rejected rather than
  silently setting an empty password.
