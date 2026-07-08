# API endpoint ↔ provider mapping

How the MXroute REST API (<https://api.mxroute.com/docs>) maps onto this
provider's resources and data sources. The relationship is not one-to-one: a
single resource can drive several endpoints (e.g. `mxroute_domain` also
toggles mail hosting), and some endpoints are deliberately unused (the
`list` GETs, since most reads use a single-object GET or a list-filter).

This file is **hand-maintained** — update it when an endpoint or a
resource/data source changes. It is the design map behind the generated
per-resource docs under `docs/`.

## Resources

| Resource | Create | Read | Update | Delete |
|----------|--------|------|--------|--------|
| `mxroute_domain` | `POST /domains` | `GET /domains/{domain}` | `PATCH /domains/{domain}/mail-status` (mail hosting) | `DELETE /domains/{domain}` |
| `mxroute_email_account` | `POST /domains/{domain}/email-accounts` | `GET /domains/{domain}/email-accounts/{user}` | `PATCH /domains/{domain}/email-accounts/{user}` | `DELETE /domains/{domain}/email-accounts/{user}` |
| `mxroute_forwarder` | `POST /domains/{domain}/forwarders` | `GET /domains/{domain}/forwarders` (list, filter by alias) | — (RequiresReplace) | `DELETE /domains/{domain}/forwarders/{alias}` |
| `mxroute_pointer` | `POST /domains/{domain}/pointers` | `GET /domains/{domain}/pointers` (list, filter by pointer) | — (RequiresReplace) | `DELETE /domains/{domain}/pointers/{pointer}` |
| `mxroute_catch_all` | `PATCH /domains/{domain}/catch-all` | `GET /domains/{domain}/catch-all` | `PATCH /domains/{domain}/catch-all` | `PATCH /domains/{domain}/catch-all` (reset to `fail`) |
| `mxroute_spam_settings` | `PATCH /domains/{domain}/spam/settings` | `GET /domains/{domain}/spam/settings` | `PATCH /domains/{domain}/spam/settings` | — (no endpoint; state-only) |
| `mxroute_spam_blacklist_entry` | `POST /domains/{domain}/spam/blacklist` | `GET /domains/{domain}/spam/blacklist` (list, filter by entry) | — (RequiresReplace) | `DELETE /domains/{domain}/spam/blacklist/{entry}` |
| `mxroute_spam_whitelist_entry` | `POST /domains/{domain}/spam/whitelist` | `GET /domains/{domain}/spam/whitelist` (list, filter by entry) | — (RequiresReplace) | `DELETE /domains/{domain}/spam/whitelist/{entry}` |
| `mxroute_reseller_package` | `POST /reseller/packages` | `GET /reseller/packages/{name}` | `PATCH /reseller/packages/{name}` | `DELETE /reseller/packages/{name}` |
| `mxroute_reseller_user` | `POST /reseller/users` | `GET /reseller/users/{username}` | `PATCH /reseller/users/{username}` · `PATCH /reseller/users/{username}/package` · `POST /reseller/users/{username}/suspend` · `POST /reseller/users/{username}/unsuspend` | `DELETE /reseller/users/{username}` |

## Data sources

| Data source | Read |
|-------------|------|
| `mxroute_domain` | `GET /domains/{domain}` (one domain) |
| `mxroute_domains` | `GET /domains` (all domain names) |
| `mxroute_dns` | `GET /domains/{domain}/dns` |
| `mxroute_email_accounts` | `GET /domains/{domain}/email-accounts` (a domain's mailboxes) |
| `mxroute_quota` | `GET /quota` |
| `mxroute_email_quota` | `GET /quota/email` |
| `mxroute_verification_key` | `GET /verification-key` |
| `mxroute_reseller_packages` | `GET /reseller/packages` (all package names) |
| `mxroute_reseller_users` | `GET /reseller/users` (all usernames) |
| `mxroute_reseller_package` | `GET /reseller/packages/{name}` (one package) |
| `mxroute_reseller_user` | `GET /reseller/users/{username}` (one user) |
| `mxroute_forwarders` | `GET /domains/{domain}/forwarders` (a domain's forwarders) |
| `mxroute_pointers` | `GET /domains/{domain}/pointers` (a domain's pointers) |
| `mxroute_spam_blacklist` | `GET /domains/{domain}/spam/blacklist` (a domain's blacklist) |
| `mxroute_spam_whitelist` | `GET /domains/{domain}/spam/whitelist` (a domain's whitelist) |

The **plural** data sources (`mxroute_domains`, `mxroute_email_accounts`,
`mxroute_reseller_packages`, `mxroute_reseller_users`) list every object; the
**singular** counterpart (`mxroute_domain`, the `mxroute_email_account`
resource, …) reads one by key. `GET /domains`, `/reseller/packages`, and
`/reseller/users` return only **names**; `GET …/email-accounts` returns full
mailbox objects.

## Non-obvious mappings

- **Singletons** (`mxroute_catch_all`, `mxroute_spam_settings`) — a per-domain
  setting with no create/delete verb. `Create` and `Update` are both the
  `PATCH`; `Read` is the `GET`. `catch_all` `Delete` resets the policy to
  `fail` (the API default); `spam_settings` has **no** DELETE endpoint, so its
  `Delete` only drops the resource from Terraform state (the domain's settings
  are left as-is).
- **List-filter reads** (`mxroute_forwarder`, `mxroute_pointer`,
  `mxroute_spam_blacklist_entry`, `mxroute_spam_whitelist_entry`) — the API has
  no single-object GET for these, so `Read` fetches the whole list endpoint and
  filters for this entry by its key. These have no `Update` (any change is a
  `RequiresReplace` = delete + create).
- **`mxroute_domain` update** is only the `mail-status` toggle — `mail_hosting`
  is the sole mutable attribute; `domain` itself is `RequiresReplace`.
- **`Domain.pointers` shape** — the OpenAPI spec declares `pointers` as an
  array of strings, but the live `GET /domains/{domain}` returns it as an
  **object keyed by pointer name** once the domain has any pointer (an empty
  domain may return `[]`). The `Domain` model decodes both shapes tolerantly
  into the list of pointer names (`models.go` `UnmarshalJSON`); the object's
  keys are the names. The dedicated `GET /domains/{domain}/pointers` endpoint
  is unaffected — it returns the spec's array of `DomainPointer` objects.
- **`mxroute_reseller_user`** spans several endpoints: the base
  `POST`/`GET`/`PATCH`/`DELETE` plus `PATCH …/package` (change package),
  `POST …/suspend` and `POST …/unsuspend` (the `suspended` attribute).

## Endpoint coverage

**Every** MXroute API endpoint is now mapped to a resource or data source —
the four former `list`-GET gaps are covered by the plural data sources above.
Keep it that way: when the API gains an endpoint, add the resource/data source
and a row here.
