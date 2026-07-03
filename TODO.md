# TODO

## Provider Setup

- [ ] Verify against the live account (via acceptance tests) the caveats the
  fan-out flagged in code comments: `/quota` + `/quota/email` enveloping
  (may be unwrapped), and the spam **blacklist** GET response shape (assumed
  `[]string` like the whitelist). Reseller user/package are unverifiable
  without a reseller account.
- [x] GitHub issue templates (`.github/ISSUE_TEMPLATE/`): bug report + feature
  request forms + `config.yml`.
- [ ] Promote the Go security/complexity tooling added here (`gosec`,
  `gocyclo`, the `govulncheck` CI job) into the global `go.md` /
  `golangci-lint.md` so every Go repo inherits it.
- [ ] Regenerate docs with tfplugindocs (blocked on the `generate` fix below);
  add `examples/` per resource.

## Repo Setup

- [ ] Release signing + Registry (see [RELEASING.md](RELEASING.md)): GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets, Registry registration; tag `v0.1.0` when the first resource lands.
- [ ] Docs generation: the `generate` CI check needs a real `terraform` (the docker wrapper can't reach `../examples`); wire one, then make `generate` required. Advisory for now.
