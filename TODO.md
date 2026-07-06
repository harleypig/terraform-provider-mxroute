# TODO

Open, actionable work only. Decisions already made live in
[`adr/`](adr/README.md), not here.

## Registry docs

- [ ] Flesh out the provider **Overview** on the registry.terraform.io
  landing page. It renders `docs/index.md`, generated from
  `templates/index.md.tmpl` (plus the provider schema) — currently only a
  two-sentence blurb before Requirements/Example Usage. Expand the template's
  prose into a proper overview: what the provider manages (the
  resource/data-source catalog by area — domains, mailboxes,
  forwarders/pointers, catch-all, spam, reseller), the auth model (three
  headers with `MXROUTE_*` env-var fallback), the write-only password
  handling, and links to the MXroute API/docs. Then regenerate `docs/index.md`
  (`make generate`) and confirm it via CI `generate`.

## Live review (blocked on a verified test domain)

Every item here needs live-account verification, which needs a **verified
throwaway test domain** — so the first item is the enabler; do it first, then
the rest can be exercised via `make testacc`. Items marked **⟨reseller⟩** also
need reseller API access, which this account does not have.

- [ ] **Set up a verified throwaway test domain (`throwaway.harleypig.dev`)**
  — the enabler for everything below. MXroute rejects adding any new domain
  (HTTP 422 `Domain verification required`) until a DNS TXT ownership record
  proves it, so a fresh throwaway can't be stood up in-test. Steps: add the
  subdomain in the MXroute panel/API, publish the required TXT record (its DNS
  lives in Linode via harleydev's `domains/` config), complete verification,
  then set `MXROUTE_TEST_DOMAIN` to it — locally (for `make testacc`) and as a
  CI secret (the `Acceptance Tests` job currently **skips** because it's
  unset). The `testAccTestDomain` guard forbids `harleypig.com`, so a
  dedicated throwaway is the only way to exercise the domain-managing tests.
- [ ] Verify the `/quota` + `/quota/email` response enveloping (they may be
  unwrapped) and the spam **blacklist** GET response shape (assumed `[]string`
  like the whitelist). The demon provider decodes `/quota` **unenveloped**,
  corroborating that read (its `/quota` endpoints were also 500ing upstream at
  comparison time).
- [ ] Confirm the documented `ssl_enabled` behavior: the attribute description
  states it is `false` immediately after domain create and flips to `true`
  asynchronously once AutoSSL issues the cert (inferred from DirectAdmin, not
  the MXroute API). Verify the actual timing and whether a post-create refresh
  is needed.
- [ ] Whether the `email_account` CREATE body accepts `limit` (we now send
  it): confirm a `limit` set at create round-trips rather than triggering a
  provider-inconsistent-result error.
- [ ] Whether the API requires `@`/`+` percent-encoded in path segments (e.g.
  a spam entry or forwarder alias with `+`). `pathSeg` uses `url.PathEscape`,
  which encodes `*` and `/ # ? space` but leaves `@`/`+` as RFC-valid pchar. If
  a live DELETE of an entry containing `@`/`+` misses, switch `pathSeg` to a
  stricter encoder (encode those too). Exercise with a `foo+bar@x` alias/entry.
- [ ] **⟨reseller⟩** `mxroute_reseller_user` `username` bounds — the last
  spec-audit refinement. Add `stringvalidator.LengthBetween(1, 10)` +
  `RegexMatches(^[a-z0-9_]+$)` (`reseller_user_resource.go`). Deferred: the
  constraint is prose-only in the spec `description` ("1-10 chars, lowercase
  letters, numbers, underscores") with no `minLength`/`maxLength`/`pattern`
  keyword — confirm the exact bounds live before enforcing (openapi.yaml:1191).
- [ ] **⟨reseller⟩** Whether the reseller API accepts a per-user quota PATCH —
  if not, our settable `mxroute_reseller_user` quota input is a misleading
  no-op and should become computed (as demon models it).
