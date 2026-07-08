# 4. Don't run live acceptance tests in CI (run them locally on demand)

- Status: accepted
- Date: 2026-07-08

## Context

The domain-lifecycle acceptance tests need `MXROUTE_TEST_DOMAIN` set to create
and destroy real domains, and all live tests need the MXroute credentials. The
`Acceptance Tests` CI job skips them without those, so it currently runs
reporting-only. Setting `MXROUTE_TEST_DOMAIN` (and the credentials) as CI
secrets was considered, so every PR would exercise real domain create/destroy.

The live account is the same one hosting `harleypig.com` **production mail**.
Wiring the secrets into CI means every PR run mutates that account
automatically and unattended.

harleydev's `bin/mxroute-provider-testacc` now wraps `make testacc` into one
guarded command. It bakes in the run-mechanics that the first live run
(2026-07-09) surfaced: a **real** terraform binary via
`TF_ACC_TERRAFORM_PATH` (the PATH terraform is a docker wrapper that breaks
plugin-testing's reattach), `MXROUTE_TEST_DOMAIN=harleypig.dev`, the
`{harleypig.dev, harleydev.com}` allow-list guard, credential loading via
`set_env`, and a confirmation prompt before anything touches the account.

## Decision

**Don't add `MXROUTE_TEST_DOMAIN` or the MXroute credentials as CI secrets.**
Keep the `Acceptance Tests` CI job reporting-only (it skips the live tests
without them). Run the live acceptance suite **deliberately, on demand**, via
harleydev's `bin/mxroute-provider-testacc`.

## Consequences

- CI never mutates the production-mail account unattended; a human
  confirmation gate always precedes a live run.
- Live regressions are not caught automatically per-PR — the maintainer runs
  the suite when touching provider internals or before a release. The required
  `Build` and `generate` checks still gate every PR.
- The run-mechanics (env vars, the real-terraform requirement, domain
  allow-list, creds) live in the runner, so "run the live suite" is one safe
  command instead of a remembered incantation.
- Revisit if the provider gains a dedicated **non-production** test account,
  where automated per-PR live runs would carry no risk to real mail.
