# TODO

## Acceptance testing

- [x] **RESOLVED (2026-07-08) — the test domain creates again.** The
  `HTTP 422 "Domain verification required"` block on `harleypig.dev` was an
  **MXroute-side glitch** (their Tailscale DNS acting up), cleared by MXroute
  tier-3 support — **not** a per-domain cooldown/block, and **not** anything on
  our side: our `_da-verify` TXT was published + authoritative throughout, and
  `/verification-key` matched byte-for-byte. A re-run then created the domain
  and drove the full lifecycle (coverage 24.9%→51.5%). **If it recurs:** a
  domain-add 422 while `dig TXT _da-verify-<key>.<domain>` resolves
  authoritatively **and** `TestAccVerificationKeyDataSource` passes is
  MXroute-side — escalate, don't touch DNS/DNSSEC. Full detection signature +
  incident writeup live in harleydev `e2e/mxroute.md` ("Known incident");
  harleydev's `bin/mxroute-provider-testacc` now DNS-preflights before the run.
- [ ] Grow `TF_ACC` acceptance coverage toward all resources and data sources
  (CRUD + import round-trips), scoped to **provider-internals the fabric can't
  surface** — `ImportState`, write-only `password_wo` create/rotate, error
  paths, and the read-only data sources. Complements harleydev's e2e tier
  (`e2e/mxroute.md`), which exercises the *applied* CRUD path via the
  mxroute-foundation-fabric modules. Run the suite with
  `bin/mxroute-provider-testacc` (see [TESTS.md](.claude/TESTS.md)).
  A coverage audit filled the known gaps: catch-all's `address`↔`type`
  config-validator error paths (incl. the empty-string address) and its `type`
  `OneOf`, the `spam_settings.high_score` range validator (all plan-time, so
  they run in the default CI gate), and `email_account` password **rotation**
  (`TestAccEmailAccountResource_passwordRotation`, which creates the domain so
  it waits on the block above). Every resource already has `ImportState` +
  `CheckDestroy` and every in-place-updatable resource an update step. A depth
  pass then added element-content assertions to the `forwarders` and
  `email_accounts` data sources (create a forwarder/mailbox, then assert
  `.0.alias` / `.0.email` / `.0.destinations` / `.0.quota` — not just the `.#`
  count, following the `pointer_resource_test.go` pattern) and a
  destinations-change step to `TestAccForwarderResource` exercising the
  forwarder `RequiresReplace` path. What is left is more of the same depth —
  element-content assertions on the remaining list data sources and richer
  update permutations — added as needs arise. (These live assertions were
  confirmed on the 2026-07-08 run — see below.)
- [ ] A live `make testacc` run (2026-07-08, coverage 52.7%) confirmed the
  passing assertions: `TestAccPointerResource` (the `Domain.pointers` decode
  against a live populated response), `TestAccForwardersDataSource` /
  `TestAccEmailAccountsDataSource` (list element content —
  `.0.alias`/`.0.email`/`.0.destinations`/`.0.quota`), and
  `TestAccForwarderResource` (a destinations change round-tripping through the
  `RequiresReplace` replace) all **pass**, as do every read, validator, and
  `VerificationKeyDataSource`. The run's two other failures — the `+`-in-alias
  rejection and the email-account `limit` behavior — are **resolved** (the
  `[x]` items below). A *fully* green run now blocks only on the
  spam-writes-500 bug (Features & fixes). Re-run with
  `bin/mxroute-provider-testacc` after that lands.
- [x] **RESOLVED — `+` in a forwarder alias.** The live API rejects `+` at
  create (HTTP 400 VALIDATION_ERROR — aliases allow only letters, numbers,
  dots, underscores, hyphens; must start with a letter/number), voiding the
  old `pathSeg`/`CheckDestroy` hypothesis (`+` is simply invalid; no `@`/`+`
  DELETE-path escaping needed). Added a client-side `alias` validator
  (`forwarderAliasPattern`) mirroring that charset — the OpenAPI spec leaves
  `alias` an unconstrained string, a documented spec-vs-live disparity — and
  converted `TestAccForwarderResource_plusInAlias` to a plan-time `ExpectError`
  test (runs in the default CI gate). A code TODO notes a possible upstream
  report to MXroute.
- [x] **RESOLVED (documented, not code-fixed) — `email_account.limit`.** Live
  (2026-07-08): the API ignores `limit` at create (a mailbox starts at the
  9600 default = the max), rejects a `limit` change sent without a password,
  but honors one on an update that also rotates the password. Kept `limit`
  settable and documented the create-then-rotate path (schema description, the
  `limit` ICEBOX in `email_account_resource.go`, and a CONVENTIONS *Known
  limitation*) rather than making it read-only. Tests no longer set `limit` at
  create; `_passwordRotation` now demonstrates the working set-path. An MXroute
  ticket is deferred (need more experience) — captured in the ICEBOX.

## Features & fixes

- [ ] **Bug: spam writes 500 on a fresh domain.** `mxroute_spam_settings`,
  `mxroute_spam_blacklist_entry`, and `mxroute_spam_whitelist_entry` all fail
  `HTTP 500 Failed to update spam settings/list` against a just-created domain
  (both spam data sources pass, so the GET shapes are fine). **Confirmed still
  reproducing on the 2026-07-08 live run** once the 422 cleared — the three
  spam writes fail while reads, validators, and `VerificationKeyDataSource`
  pass. Investigate fresh-vs-established domain (a read against harleypig.com
  is safe anytime); open an MXroute ticket if it reproduces generally. Blocks
  the spam-entry DELETE path (and its `@`/`+` encoding) until the creates
  succeed.
