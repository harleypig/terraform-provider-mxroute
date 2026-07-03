# TODO

## Provider Setup

- [x] Wire the provider: `Configure` builds the `Client` and passes it as
  `ResourceData`/`DataSourceData`; provider schema takes
  `server`/`username`/`api_key` with `MXROUTE_*` env fallback (replaces the
  template's `endpoint`/`http.DefaultClient` scaffolding).
- [x] `mxroute_domain` data source + resource (`/domains`, `/domains/{d}`) —
  create/read/delete + import; the proven template the remaining resources
  (and any fan-out workflow) follow.
- [ ] `mxroute_email_account` resource — write-only password (`WriteOnly` + `*_wo_version` trigger).
- [ ] `mxroute_forwarder`, `mxroute_pointer` resources; `mxroute_dns` data source.
- [x] Remove the remaining template scaffolding (function / action / ephemeral
  resource + tests + docs) — dropped, as the provider now requires credentials
  and MXroute has no such surface planned.
- [ ] Acceptance tests against the live account (`TF_ACC`), gated out of
  default CI — `mxroute_domain` covered; add per new resource.
- [ ] Regenerate docs with tfplugindocs (blocked on the `generate` fix below);
  add `examples/` per resource.

## Repo Setup

- [ ] Release signing + Registry (see [RELEASING.md](RELEASING.md)): GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets, Registry registration; tag `v0.1.0` when the first resource lands.
- [ ] Docs generation: the `generate` CI check needs a real `terraform` (the docker wrapper can't reach `../examples`); wire one, then make `generate` required. Advisory for now.
