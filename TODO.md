# TODO

## Acceptance testing

- [ ] Grow `TF_ACC` acceptance coverage as needs arise, scoped to
  **provider-internals the fabric can't surface** (`ImportState`, write-only
  `password_wo`, error paths, the read-only data sources) — it complements
  harleydev's applied-CRUD e2e tier (`e2e/mxroute.md`). Breadth is done (every
  resource has `ImportState` + `CheckDestroy`; every in-place-updatable
  resource an update step); what's left is depth — element-content assertions
  on the remaining list data sources and richer multi-attribute update
  permutations. Run with `bin/mxroute-provider-testacc` (see
  [TESTS.md](.claude/TESTS.md)).
- [ ] Live-confirm `TestAccDomainResource_unverified422` on the next
  `make testacc`. It applies `mxroute_domain` to an unverified domain and
  `ExpectError`s MXroute's `HTTP 422 "Domain verification required"` (native
  `terraform test` can't assert a provider apply-error, so this lives here,
  not in harleydev's e2e suite). Needs a genuinely unverified, allow-listed
  domain: `MXROUTE_TEST_UNVERIFIED_DOMAIN=harleydev.com` (skips when unset,
  never the live domain).
- [ ] Add a `TESTARGS` (or `-run` name-filter) passthrough to the `testacc`
  make target so a scoped live run of specific acceptance tests is possible
  without hand-rolling the env (the target hardcodes `./...`, so probing one
  resource live means bypassing `bin/mxroute-provider-testacc` to
  `source bin/set_env` + `TF_ACC=1 TF_ACC_TERRAFORM_PATH=… go test -run …`).
  Consider a matching name-filter flag on harleydev's runner.

## Features & fixes

- [ ] **Bug: spam writes 500 on a fresh domain.** `mxroute_spam_settings`,
  `mxroute_spam_blacklist_entry`, and `mxroute_spam_whitelist_entry` all fail
  `HTTP 500 Failed to update spam settings/list` against a just-created domain
  (both spam data sources pass, so the GET shapes are fine) — confirmed
  reproducing on the 2026-07-08 live run, where these three writes were the
  only failures (reads, validators, and `VerificationKeyDataSource` pass).
  Investigate fresh-vs-established domain (a read against harleypig.com is
  safe anytime); open an MXroute ticket if it reproduces generally. Blocks the
  spam-entry DELETE path (and its `@`/`+` encoding) until the creates succeed.
