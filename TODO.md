# TODO

## Acceptance testing

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
- [ ] **Unverified-domain 422 negative** (`TestAccDomainResource_unverified422`)
  — BUILT: scenario 6 of harleydev's `e2e/mxroute.md`, rerouted here because
  native `terraform test` can't assert a provider apply-error
  (`expect_failures` catches only condition/validation failures). Applies
  `mxroute_domain` on an unverified domain and `ExpectError`s MXroute's
  `HTTP 422 "Domain verification required"`. Needs a genuinely unverified,
  allow-listed domain — set `MXROUTE_TEST_UNVERIFIED_DOMAIN=harleydev.com`
  (skips when unset; never the live domain). Awaits live confirmation on the
  next `make testacc`.
- [ ] A live `make testacc` run (2026-07-08, coverage 52.7%) confirmed the
  depth and prior assertions pass: `TestAccPointerResource` (the
  `Domain.pointers` decode against a live populated response),
  `TestAccForwardersDataSource` / `TestAccEmailAccountsDataSource` (list
  element content), and `TestAccForwarderResource` (a destinations change
  through the `RequiresReplace` replace), alongside every read, validator, and
  `VerificationKeyDataSource`. A *fully* green run now blocks only on the
  spam-writes-500 bug (Features & fixes); re-run with
  `bin/mxroute-provider-testacc` after that lands.
- [ ] Add a `TESTARGS` (or `-run` name-filter) passthrough to the `testacc`
  make target so a scoped live run of specific acceptance tests is possible
  without hand-rolling the env (the target hardcodes `./...`, so probing one
  resource live means bypassing `bin/mxroute-provider-testacc` to
  `source bin/set_env` + `TF_ACC=1 TF_ACC_TERRAFORM_PATH=… go test -run …`).
  Consider a matching name-filter flag on harleydev's runner.
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
