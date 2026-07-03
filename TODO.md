# TODO

## Provider Setup

- [x] `models.go` — canonical types for every API component schema + inline
  response shape; client hardening (all 8 error codes tested, rate-limit
  retry via `Retry-After`/`X-RateLimit-Reset`, `IsConflict`/`IsRateLimited`).
- [ ] Fan out: refactor the 5 existing resources onto `models.go` and create
  the new modules — catch-all, spam (settings/blacklist/whitelist), quota +
  email-quota + verification-key data sources, reseller users/packages, and a
  writable `mail_hosting` on `mxroute_domain` (`PATCH mail-status`).
- [ ] GitHub issue templates (`.github/ISSUE_TEMPLATE/`): bug report + feature
  request forms + `config.yml`.
- [ ] Regenerate docs with tfplugindocs (blocked on the `generate` fix below);
  add `examples/` per resource.

## Repo Setup

- [ ] Release signing + Registry (see [RELEASING.md](RELEASING.md)): GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets, Registry registration; tag `v0.1.0` when the first resource lands.
- [ ] Docs generation: the `generate` CI check needs a real `terraform` (the docker wrapper can't reach `../examples`); wire one, then make `generate` required. Advisory for now.
