# Icebox

Deferred "not now / maybe-someday" work — revisited **only on the trigger
noted**, not on the general [`TODO.md`](TODO.md) cadence. Per the maintainer's
`code-style.md` (the `ICEBOX:` convention) and `todo.md` (deferred work is not
a TODO item). Each entry carries an `ICEBOX:` tag so a
`grep -rn "ICEBOX:"` scan for prior deferred decisions surfaces it.

## Reseller support

`ICEBOX:` reseller — **trigger: this account (or any wired-up account) gains
reseller API access.** The harleypig account has no reseller privileges, so
none of the reseller surface can be built, verified, or exercised live (the
endpoints return `HTTP 403`). Revisit the whole cluster when a
reseller-capable account is available.

- **`mxroute_reseller_user` `username` bounds.** Add
  `stringvalidator.LengthBetween(1, 10)` + `RegexMatches(^[a-z0-9_]+$)` in
  `reseller_user_resource.go`. The constraint is prose-only in the spec
  `description` ("1-10 chars, lowercase letters, numbers, underscores") with
  no `minLength`/`maxLength`/`pattern` keyword (`openapi.yaml:1191`) — confirm
  the exact bounds live before enforcing, so the validator can't reject a
  username the API accepts.
- **Per-user quota PATCH.** Determine whether the reseller API accepts a
  per-user quota PATCH. If it does not, the settable `mxroute_reseller_user`
  quota input is a misleading no-op and should become `Computed` (as
  `demon-tf-provider-mxroute` models it).
