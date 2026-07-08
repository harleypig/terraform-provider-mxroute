# TODO

Open, actionable work only. Decisions already made live in
[`adr/`](adr/README.md), not here.

## Live review

Every item here needs **live-account** verification — but they split by what
kind of live access, and only the first group is blocked on the test domain.
**None of them block harleydev's mail migration:** the migration never
creates or deletes a *domain* (harleypig.com is already on the account and
verified), and its own safe, account-side applies naturally exercise the
second group's slices.

### Blocked on the verified test domain (domain-lifecycle tests)

- [ ] **Set up the verified test domain — decided: `harleypig.dev`**
  (`MXROUTE_TEST_DOMAIN=harleypig.dev`; harleydev's e2e design resolved the
  throwaway question — harleypig.dev is the designated repeating-test domain,
  and MXroute's required `mail`/`webmail` subdomains are the throwaway records
  exercised; see harleydev `e2e/mxroute.md`). This is the enabler for the
  domain-lifecycle items, **shared with harleydev's e2e tier** (its Phase 1).
  MXroute rejects adding any new domain (HTTP 422 `Domain verification
  required`) until a DNS TXT ownership record proves it, so a fresh domain
  can't be stood up in-test. Steps: add the domain in the MXroute panel/API,
  publish the required TXT record (its DNS lives in Linode via harleydev's
  `domains/` config), complete verification, then set
  `MXROUTE_TEST_DOMAIN=harleypig.dev` — locally (for `make testacc`) and as a
  CI secret (the `Acceptance Tests` job currently **skips** because it's
  unset). The `testAccTestDomain` guard forbids `harleypig.com`, so the
  verified test domain is the only way to exercise the domain-managing tests.
  Open live question (shared with harleydev Phase 1): whether a
  destroy → recreate of a previously-verified domain re-triggers the 422
  verification requirement.
- [ ] Confirm the documented `ssl_enabled` behavior: the attribute description
  states it is `false` immediately after domain create and flips to `true`
  asynchronously once AutoSSL issues the cert (inferred from DirectAdmin, not
  the MXroute API). Verify the actual timing and whether a post-create refresh
  is needed. Needs a **fresh domain create** to observe — test-domain only.
- [ ] Full `TF_ACC` acceptance coverage across all resources and data sources
  (CRUD + import round-trips), building on the verified-test-domain enabler
  above. Scope it to **provider-internals the fabric can't surface** —
  `ImportState`, write-only `password_wo` create/rotate behavior, error paths,
  and the read-only data sources. This **complements, not duplicates**,
  harleydev's e2e tier: that suite applies the
  mxroute-foundation-fabric modules against real ephemeral resources and so
  exercises the provider's *applied* CRUD path (double duty — one run tests
  both provider and fabric). **Now designed:** harleydev `e2e/mxroute.md`
  (Phase 2 of its e2e tier; overview in `e2e/README.md`; build-out tracked in
  harleydev's TODO → *e2e Testing*). Shared enabler: the verified test domain
  above (`MXROUTE_TEST_DOMAIN=harleypig.dev`).

### Needs live account only (no test domain; migration applies exercise these)

**Live outcome 2026-07-07** (harleydev's five-mailbox apply on
harleypig.com): `mxroute_email_account` **CREATE is proven** — five creates
succeeded, and write-only `password_wo` plus `quota = 0` round-trip cleanly
(a follow-up refresh/plan reports no changes). `limit` was omitted
(`omitempty`), so the limit-at-create item below remains open, as do the
`/quota`/spam-shape and `@`/`+` items.

- [ ] Verify the `/quota` + `/quota/email` response enveloping (they may be
  unwrapped) and the spam **blacklist** GET response shape (assumed `[]string`
  like the whitelist). The demon provider decodes `/quota` **unenveloped**,
  corroborating that read (its `/quota` endpoints were also 500ing upstream at
  comparison time). `/quota` is read-only — probeable against the account any
  time; the spam-entry slice gets exercised by harleydev's pre-migration spam
  setup on harleypig.com (a wrong shape is a harmless read error).
- [ ] Whether the `email_account` CREATE body accepts `limit` (sent
  `omitempty` — only when set): confirm a `limit` set at create round-trips
  rather than triggering a provider-inconsistent-result error. harleydev's
  migration mailboxes **omit** `limit`, so they never touch this path — verify
  whenever a `limit` is first set (safe on harleypig.com, account-side).
- [ ] Whether the API requires `@`/`+` percent-encoded in path segments (e.g.
  a spam entry or forwarder alias with `+`). `pathSeg` uses `url.PathEscape`,
  which encodes `*` and `/ # ? space` but leaves `@`/`+` as RFC-valid pchar. If
  a live DELETE of an entry containing `@`/`+` misses, switch `pathSeg` to a
  stricter encoder (encode those too). Exercise with a `foo+bar@x` alias/entry
  (safe on harleypig.com; migration aliases are plain names, so it's not on
  that path).

### Blocked on reseller API access (this account has none)

- [ ] **⟨reseller⟩** `mxroute_reseller_user` `username` bounds — the last
  spec-audit refinement. Add `stringvalidator.LengthBetween(1, 10)` +
  `RegexMatches(^[a-z0-9_]+$)` (`reseller_user_resource.go`). Deferred: the
  constraint is prose-only in the spec `description` ("1-10 chars, lowercase
  letters, numbers, underscores") with no `minLength`/`maxLength`/`pattern`
  keyword — confirm the exact bounds live before enforcing (openapi.yaml:1191).
- [ ] **⟨reseller⟩** Whether the reseller API accepts a per-user quota PATCH —
  if not, our settable `mxroute_reseller_user` quota input is a misleading
  no-op and should become computed (as demon models it).

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
