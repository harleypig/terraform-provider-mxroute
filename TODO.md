# TODO

Open, actionable work only. Decisions already made live in
[`adr/`](adr/README.md), not here.

**This list is the v1.0.0 gate.** The provider stays `v0.x` until every
task here is resolved — `v1` is the compatibility promise (breaking
changes then require a major bump; see the maintainer's `git.md`
*Versioning & tags*), and that promise can't be made while live-API
shapes, acceptance findings, and reseller behavior are still open. The
`0 → 1` jump is its own deliberate decision when the list empties, not
an automatic consequence.

## Live review

Every item here needs **live-account** verification — but they split by what
kind of live access, and only the first group is blocked on the test domain.
**None of them block harleydev's mail migration:** the migration never
creates or deletes a *domain* (harleypig.com is already on the account and
verified), and its own safe, account-side applies naturally exercise the
second group's slices.

### Unblocked: the verified test domain is live (domain-lifecycle tests)

**Enabler DONE 2026-07-09** — `harleypig.dev` is verified (the standing
`_da-verify` TXT lives in harleydev `domains/yaml/harleypig_dev.yml`) and
the first live `make testacc` ran against it: **19 pass / 2 skip / 9
fail**, account clean afterward (only harleypig.com on it). Domain
create/destroy cycled repeatedly within the one run — with the TXT
standing, re-adds always pass, so the destroy → recreate 422 question is
**moot in practice** (whether a TXT-less re-add would 422 was not
exercised, deliberately: the TXT never comes down). The 9 failures are
itemized under *Findings from the first live testacc run* below.

Local-run gotcha, now known: `make testacc` needs a **real** terraform
binary via `TF_ACC_TERRAFORM_PATH` (e.g. `~/.cache/tf-acc/terraform`) —
a docker-wrapped `terraform` on PATH breaks plugin-testing's
`TF_REATTACH_PROVIDERS` injection, failing every test with `Inconsistent
dependency lock file` before any API call.

- [ ] Decide whether to set `MXROUTE_TEST_DOMAIN=harleypig.dev` as a CI
  secret — the `Acceptance Tests` job currently **skips** the
  domain-lifecycle tests without it. Deliberate call, not a checkbox:
  with it set, every PR's CI creates/destroys real domains on the live
  account (the same account hosting harleypig.com production mail).
- [ ] Confirm the documented `ssl_enabled` behavior: the attribute description
  states it is `false` immediately after domain create and flips to `true`
  asynchronously once AutoSSL issues the cert (inferred from DirectAdmin, not
  the MXroute API). Verify the actual timing and whether a post-create refresh
  is needed. Needs a **fresh domain create** to observe — test-domain only
  (now available).
- [ ] Full `TF_ACC` acceptance coverage across all resources and data sources
  (CRUD + import round-trips). Scope it to **provider-internals the fabric
  can't surface** — `ImportState`, write-only `password_wo` create/rotate
  behavior, error paths, and the read-only data sources. This **complements,
  not duplicates**, harleydev's e2e tier: that suite applies the
  mxroute-foundation-fabric modules against real ephemeral resources and so
  exercises the provider's *applied* CRUD path (double duty — one run tests
  both provider and fabric). **Now designed:** harleydev `e2e/mxroute.md`
  (Phase 2 of its e2e tier; overview in `e2e/README.md`; build-out tracked in
  harleydev's TODO → *e2e Testing*). First live run done (above); clear the
  findings below, then grow coverage.

### Findings from the first live testacc run (2026-07-09)

- [x] **Bug — `Domain.pointers` decode.** `GET /domains/<domain>` returns
  `pointers` as an **object**, but the client model decoded `[]string`:
  `json: cannot unmarshal object into Go struct field Domain.pointers of
  type []string`. The pointer CREATE succeeded and `TestAccPointersDataSource`
  **passed** (the list endpoint's own shape is fine) — it was specifically the
  Domain model's field. Failed `TestAccPointerResource` at the post-apply
  refresh. **Fixed:** `Domain.UnmarshalJSON` now decodes both shapes (array of
  strings, or object keyed by pointer name) to the list of names, with a unit
  regression test and an `API-MAPPING.md` note. The exact live object shape
  (keys = names is the assumed DirectAdmin convention) still wants a live
  re-run of `TestAccPointerResource` to confirm the populated list is right.
- [ ] **Spam writes 500 on a fresh domain.** All three spam writes —
  `mxroute_spam_settings`, `mxroute_spam_blacklist_entry`,
  `mxroute_spam_whitelist_entry` — failed `HTTP 500 Failed to update spam
  settings/list` against the just-created test domain, while both spam
  **data sources passed** (GET shapes verified — closes the blacklist-shape
  question below). Investigate fresh-vs-established domain (a read against
  harleypig.com is safe anytime; harleydev's first evidence-driven spam
  entry doubles as the established-domain write test); open an MXroute
  ticket if it reproduces generally. Until resolved, the spam-entry DELETE
  path (and its `@`/`+` encoding question) stays unexercised — the creates
  never succeeded.
- [x] **Test fixture — email-account password too weak.** The API now
  enforces server-side complexity at create: `HTTP 400 VALIDATION_ERROR
  "Password does not meet minimum requirements. Use a stronger password
  with a mix of uppercase, lowercase, numbers, and special characters."`
  (`TestAccEmailAccountResource`). **Fixed:** strengthened the create fixture
  to satisfy all four classes, and documented the complexity requirement in
  the `password_wo` schema description so users see it. A plan-time complexity
  *validator* is deferred to an `ICEBOX:` note at the schema (the spec
  declares only `minLength 8`; the exact server policy — which characters are
  "special", all-four vs 3-of-4 — must be confirmed live before a client-side
  regex risks rejecting passwords the API accepts).
- [x] **Test guards — reseller data sources.** The four reseller
  data-source tests (`reseller_package`, `reseller_packages`,
  `reseller_user`, `reseller_users`) failed `HTTP 403 This endpoint requires
  reseller privileges` on a non-reseller account; the reseller *resource*
  tests already skipped cleanly. **Fixed:** added a shared
  `testAccResellerPreCheck` gate (skips unless `MXROUTE_TEST_RESELLER` opts
  in) and wired it into all four. This account has no reseller access, so
  they skip — reseller CRUD stays unexercisable until a reseller-capable
  account is wired up.

### Needs live account only (no test domain; migration applies exercise these)

**Live outcome 2026-07-07** (harleydev's five-mailbox apply on
harleypig.com): `mxroute_email_account` **CREATE is proven** — five creates
succeeded, and write-only `password_wo` plus `quota = 0` round-trip cleanly
(a follow-up refresh/plan reports no changes). `limit` was omitted
(`omitempty`), so the limit-at-create item below remains open.

**Live outcome 2026-07-09** (first testacc run): the `/quota` +
`/quota/email` **enveloping is verified** (both quota data sources passed)
and the spam blacklist/whitelist **GET shapes are verified** (both spam
data sources passed) — that item is done and pruned. The `@`/`+`
path-encoding question stays open below, narrower now: the spam-entry
DELETE path can't exercise it until the spam-write 500s (findings above)
are resolved; a `foo+bar@x` **forwarder** alias remains the available
probe.

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
