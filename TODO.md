# TODO

## Provider Setup

- [x] `mxroute_email_account` resource — write-only password (`WriteOnly` + `*_wo_version` trigger).
- [x] `mxroute_forwarder`, `mxroute_pointer` resources; `mxroute_dns` data source.
- [x] Acceptance tests against the live account (`TF_ACC`), gated out of
  default CI — all resources + data sources covered.
- [ ] Regenerate docs with tfplugindocs (blocked on the `generate` fix below);
  add `examples/` per resource.

## Repo Setup

- [ ] Release signing + Registry (see [RELEASING.md](RELEASING.md)): GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets, Registry registration; tag `v0.1.0` when the first resource lands.
- [ ] Docs generation: the `generate` CI check needs a real `terraform` (the docker wrapper can't reach `../examples`); wire one, then make `generate` required. Advisory for now.
