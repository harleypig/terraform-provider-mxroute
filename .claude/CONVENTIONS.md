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
(`go 1.25.11`, `tool` directives), so a local Go ≥ 1.21 builds it via
`GOTOOLCHAIN=auto`. CI uses `actions/setup-go` with `go-version-file: go.mod`.
Keep the pinned patch current — `govulncheck` (below) flags stdlib CVEs, and
the fix is usually a patch bump here.

## QA

- **Format + lint:** golangci-lint v2 — `golangci-lint fmt` (gofumpt/goimports)
  and `golangci-lint run`, config in `.golangci.yml`; pre-commit gates it
  (`.pre-commit-config.yaml` + `.pre-commit-config-fix.yaml`), and CI is the
  authoritative lint gate.
- **Code smell / complexity:** `staticcheck` + `gocyclo` (min-complexity 20)
  run inside golangci-lint.
- **Security:** `gosec` (SAST) inside golangci-lint; `govulncheck` (SCA / stdlib
  vuln scan, call-graph aware) as its own CI job; `gitleaks` +
  `detect-private-key` (secrets) in pre-commit.
- **Tests:** `go test -cover` (unit) + `terraform-plugin-testing` acceptance —
  see [TESTS.md](TESTS.md).
- **Docs:** `tfplugindocs` via `go generate`, kept current.

## Merge policy & versioning

- `master` is **PR-only** (server-side ruleset + the local `no-commit-to-branch`
  hook).
- **Versioning:** semver `vX.Y.Z`, with the **MAJOR aligned to the MXroute
  API's major** (see *Versioning & tagging* below). A tag triggers the
  GoReleaser + GPG release that publishes to the Terraform Registry. Cut tags
  with the `release-tag` skill (annotated, at the merge commit).

## Versioning & tagging

The provider is versioned **`repo`-scope semver** (one `vX.Y.Z` for the whole
provider; the global `git.md` › *Versioning & tags* method), with one
repo-specific **bump policy**: the provider's **MAJOR tracks the MXroute API's
MAJOR**. This keeps the registry-required semver contract while making a tag
legibly signal which API generation it targets.

The API declares its own version in its OpenAPI `info.version`
(currently **`1.0.0`** → API major **1**; verify at
`https://api.mxroute.com/openapi.json`, field `info.version`).

- **MAJOR = API major.** A release targeting API `1.x` carries major `1`. When
  MXroute ships a **breaking** API `2.0.0`, the provider's next release is
  `2.0.0`. The API's own backward-compatible minor/patch (`1.0 → 1.1`) do
  **not** force a provider major bump.
- **MINOR / PATCH move on the provider's own cadence**, within a major:
  - **MINOR** — a new resource/data source or feature, including support the
    provider adds for a backward-compatible API addition.
  - **PATCH** — a provider fix, dependency bump, or docs-only release. These
    ship **without** any API version change — the reason literal API-lockstep
    is rejected (it would forbid an independent provider fix).
- **Alpha (now) — `v0.y.z`.** The provider is pre-stable, so it stays on
  `v0.y.z` (breakage expected, loose `y.z`; `git.md`) and the targeted API
  version is **documented**, not yet encoded in the major. The deliberate
  **`0 → 1` stability jump** is when the provider adopts the API's current
  major as its own — the first stable tag is `1.0.0`, declared to target API
  `1.x`.
- **Every release documents its targeted API version** — a one-line
  `Compatibility: targets MXroute API 1.x` in the release notes / changelog —
  so the tag ↔ API relationship stays explicit even while alpha encodes only
  the minor.

Tags are **annotated**, cut at the merge commit on `master`, via the
`release-tag` skill. See [RELEASING.md](../RELEASING.md) for the release
mechanics.
