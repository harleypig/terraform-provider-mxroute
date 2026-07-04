# TODO

## Provider Setup

- [ ] Verify against the live account (via acceptance tests) the caveats the
  fan-out flagged in code comments: `/quota` + `/quota/email` enveloping
  (may be unwrapped), and the spam **blacklist** GET response shape (assumed
  `[]string` like the whitelist). Reseller user/package are unverifiable
  without a reseller account.
- [ ] Promote the Go security/complexity tooling added here (`gosec`,
  `gocyclo`, the `govulncheck` CI job) into the global `go.md` /
  `golangci-lint.md` so every Go repo inherits it.

## Repo Setup

- [ ] Release signing + Registry (see [RELEASING.md](RELEASING.md)): GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets, Registry registration; tag `v0.1.0` when the first resource lands.
- [ ] Make the `generate` check **required**: it now passes (CI already has a
  real terraform via `setup-terraform`; the failures were only stale docs, now
  regenerated). Add it to the branch-ruleset required checks. Local regeneration
  needs a real terraform on PATH — the docker-wrapper `bin/terraform` can't
  reach `../examples` across the mount boundary.
