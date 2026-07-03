# TODO

## Provider Setup

- [ ] API client (`internal/provider/client.go`): 3-header auth, `{success,data}` envelope unwrap, error mapping.
- [ ] `mxroute_domain` data source + resource (`/domains`, `/domains/{d}`).
- [ ] `mxroute_email_account` resource — write-only password (`WriteOnly` + `*_wo_version` trigger).
- [ ] `mxroute_forwarder`, `mxroute_pointer` resources; `mxroute_dns` data source.
- [ ] Replace the template's example resource/data source/function/action.
- [ ] Acceptance tests against the live account (`TF_ACC`), gated out of default CI.
- [ ] Regenerate docs with tfplugindocs; add `examples/` per resource.

## Repo Setup

- [ ] Release signing + Registry (see [RELEASING.md](RELEASING.md)): GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets, Registry registration; tag `v0.1.0` when the first resource lands.
- [ ] Docs generation: the `generate` CI check needs a real `terraform` (the docker wrapper can't reach `../examples`); wire one, then make `generate` required. Advisory for now.
