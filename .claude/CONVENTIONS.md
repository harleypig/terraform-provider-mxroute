# terraform-provider-mxroute Conventions

Repo-specific conventions. The global `~/.claude/` config carries everything
generic (git/gh, code style, Go tooling via `go.md` + `golangci-lint.md`, the
QA dimensions, and the `terraform-provider-patterns` skill). This file records
only what is specific to **this** repo.

## What this is

An MIT-licensed Terraform provider for MXroute email hosting, built on
`terraform-plugin-framework` (protocol v6). Registry address
`registry.terraform.io/harleypig/mxroute`; module
`github.com/harleypig/terraform-provider-mxroute`.

## Layout

- `main.go` — `providerserver.Serve`.
- `internal/provider/` — the provider, resources, data sources, and the API
  client (`client.go`).
- `examples/`, `templates/`, `docs/` — tfplugindocs inputs/outputs.

## The MXroute API

- Base `https://api.mxroute.com`; flat REST (`/domains`, `/domains/{d}/…`).
- Auth: three headers — `X-Server`, `X-Username`, `X-API-Key`.
- Every response is wrapped `{"success":bool,"data":…,"error":{…}}`; the client
  unwraps `data` and maps `success:false` to a Go error.
- Provider config takes the three values (env fallback to the `MXROUTE_*`
  vars); the API key attribute is `Sensitive`.

## Resource conventions

Follow the `terraform-provider-patterns` skill:

- Declare an explicit `Computed` `id`; implement `ImportState` on every
  resource.
- **Write-only secrets:** a create-only value (a mailbox password) is a
  `WriteOnly` attribute paired with a `*_wo_version` trigger — never a
  `Sensitive` field that persists to state.
- Plan modifiers: `RequiresReplace` where the API can't update in place;
  `UseStateForUnknown` for stable computed values.

## Toolchain & reproducibility

Per `go.md`: the Go version and dev tools are pinned in `go.mod`
(`toolchain go1.25.8`, `tool` directives), so a local Go ≥ 1.21 builds it via
`GOTOOLCHAIN=auto`. CI uses `actions/setup-go` with `go-version-file: go.mod`.

## QA

- **Format + lint:** golangci-lint v2 — `golangci-lint fmt` (gofumpt/goimports)
  and `golangci-lint run`, config in `.golangci.yml`; pre-commit gates it
  (`.pre-commit-config.yaml` + `.pre-commit-config-fix.yaml`), and CI is the
  authoritative lint gate.
- **Tests:** `go test` (unit) + `terraform-plugin-testing` acceptance — see
  [TESTS.md](TESTS.md).
- **Docs:** `tfplugindocs` via `go generate`, kept current.

## Merge policy & versioning

- `master` is **PR-only** (server-side ruleset + the local `no-commit-to-branch`
  hook).
- **Versioning:** semver `vX.Y.Z`; `v0.y.z` is alpha (breakage expected, loose
  `y.z`). A tag triggers the GoReleaser + GPG release that publishes to the
  Terraform Registry. Cut tags with the `release-tag` skill (annotated, at the
  merge commit).
