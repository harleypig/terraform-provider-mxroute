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

- [ ] Wire dotfiles conventions: pre-commit (Go hooks + generic + markdownlint), `.claude/`, align `.golangci.yml` to v2.
- [ ] Branch protection (server ruleset + `no-commit-to-branch`).
- [ ] Release: GoReleaser + GPG, publish to the Terraform Registry.
