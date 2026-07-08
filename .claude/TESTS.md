# terraform-provider-mxroute Test Layout

The global `testing.md` carries the bar (success + failure paths, a regression
test per bug); the `terraform-provider-patterns` skill carries the acceptance-
test recipe. This file records what exists here.

## Two tiers

1. **Unit tests** (`*_test.go` beside the code) — table-driven, no network.
   The default gate; run in CI on every PR. Cover the client's envelope
   handling, schema/plan logic, and error mapping.
2. **Acceptance tests** (`TF_ACC=1`, `terraform-plugin-testing`) — stand up
   **real** MXroute resources against the **live account**. They mutate real
   state; run manually (`make testacc`), **never** in the default CI gate.

A **complementary third tier lives out of repo**: harleydev's e2e suite
(harleydev `e2e/mxroute.md`) applies the mxroute-foundation-fabric modules
against real ephemeral resources — the provider's *applied* CRUD path through
module composition. This repo's acceptance tests stay scoped to
provider-internals the fabric can't surface (ImportState, write-only
`password_wo`, error paths, data sources); the shared enabler is the verified
test domain, `MXROUTE_TEST_DOMAIN=harleypig.dev` (see TODO → *Live review*).
Recipe skills: `terraform-provider-patterns` (this repo's acceptance tier),
`terraform-e2e-patterns` (the apply-mode e2e tier).

## Acceptance-test credentials & safety

- Credentials come from the harleydev `bin/set_env` (the three `MXROUTE_*` env
  vars → `X-Server` / `X-Username` / `X-API-Key`). `TF_ACC=1` is required, so a
  plain `go test` never touches the account.
- Every acceptance test sets **`CheckDestroy`** to confirm cleanup.
- The account has one domain (`harleypig.com`); tests that create mailboxes /
  forwarders / pointers must tear them down and must not disturb live mail.

## Running

```sh
go test ./...      # unit (credential-free)
make testacc       # acceptance (TF_ACC=1; needs `. bin/set_env` from harleydev)
```

For the acceptance suite, prefer harleydev's
**`bin/mxroute-provider-testacc`** over a bare `make testacc`: it bakes in the
run-mechanics — a **real** terraform binary via `TF_ACC_TERRAFORM_PATH` (the
PATH `terraform` is a docker wrapper that breaks plugin-testing's reattach),
`MXROUTE_TEST_DOMAIN=harleypig.dev`, the `{harleypig.dev, harleydev.com}`
domain allow-list guard, credential loading, and a confirmation gate before it
touches the live account. Live tests run **on demand** this way, never in CI —
see [ADR 0004](../adr/0004-no-live-acceptance-tests-in-ci.md).
