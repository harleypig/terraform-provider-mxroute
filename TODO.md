# TODO

Open, actionable work only. Decisions already made live in
[`adr/`](adr/README.md), not here.

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
- [ ] Full `TF_ACC` acceptance coverage across all resources and data sources
  (CRUD + import round-trips), building on the verified-test-domain enabler
  above. Scope it to **provider-internals the fabric can't surface** —
  `ImportState`, write-only `password_wo` create/rotate behavior, error paths,
  and the read-only data sources. This **complements, not duplicates**,
  harleydev's integration tier: that suite applies the
  mxroute-foundation-fabric modules against real ephemeral resources and so
  exercises the provider's *applied* CRUD path (double duty — one run tests
  both provider and fabric). See harleydev's TODO → *Account IaC* → "Full e2e /
  integration testing for the `mxroute` stack". Shared enabler: the verified
  throwaway test domain.

## Documentation

- [ ] Provider-native versions of the mff usage guides. The
  [mxroute-foundation-fabric][mff] repo ships two how-to guides in its `docs/`
  — `quick-start.md` (stand up a domain + mailbox + forwarder) and
  `email-management.md` (mailboxes, forwarders, spam filtering) — but they are
  written around the **fabric modules** (`source =
  ".../mxroute-foundation-fabric//modules/..."`, the module
  `email_accounts = {…}` map shape, links to module READMEs). Author
  provider-native equivalents under `templates/guides/*.md.tmpl` (rendered to
  `docs/guides/` by `tfplugindocs generate`, so they appear as guide pages on
  the registry alongside the Overview) that use the **resources directly** —
  `mxroute_domain`, `mxroute_email_account`, `mxroute_forwarder`,
  `mxroute_catch_all`, `mxroute_spam_*` — and drop the dev-override /
  not-yet-on-Registry note now that the provider is published. Much of the
  surrounding prose carries over largely unchanged: the scope caveat (the
  provider manages the MXroute **account side** via the API, **not** DNS), the
  MX / SPF / DKIM / DMARC record table, and the client-port list. Judgment
  call while writing: whether these live better as provider guides here or stay
  module-flavored in mff — the module map is a genuinely different UX from raw
  resources, so a straight copy may not fit.

[mff]: https://github.com/harleypig/mxroute-foundation-fabric
