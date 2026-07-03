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
