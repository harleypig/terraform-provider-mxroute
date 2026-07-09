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

**Known limitation — no SSL/certificate management.** The API exposes no
operation to request or issue a TLS certificate; `ssl_enabled` is a
response-only boolean on the domain (verified against `api/openapi.yaml` — no
POST/PATCH or AutoSSL trigger). Certificates are provisioned out-of-band (the
MXroute/DirectAdmin panel), so `mxroute_domain.ssl_enabled` is read-only
status the provider can only report — don't re-attempt building SSL management
against this API.

**Known limitation — `email_account.limit` is read-only.** Confirmed live: the
API does **not** reliably honor a user-set `limit`. It **ignores** the value
at create (a new mailbox is always the `9600` default, which is also the max),
**rejects** a change sent without a password (HTTP 400), and even a change
sent **with** a password rotation is applied only **intermittently** — the
same path returned the requested `5000` twice, then the `9600` default. Rather
than ship a settable attribute whose `apply` randomly fails, the provider
exposes `limit` **read-only** (dropped from the create/update bodies; the
server value is read back). `quota` has none of these quirks. Don't re-attempt
making `limit` settable unless MXroute fixes the write behaviour. TODO: file
an MXroute bug report for the unreliable/undocumented `limit` writes (see the
`limit` comment in `email_account_resource.go`).

**Known limitation — spam writes 500.** Confirmed live (2026-07-08 and the v1
run): every spam **write** — `mxroute_spam_settings` (PATCH `high_score`), and
`mxroute_spam_blacklist_entry` / `mxroute_spam_whitelist_entry` (POST `entry`)
— fails `HTTP 500 "Failed to update spam settings/list"` on the test domain,
while both spam **data sources** (the GETs) succeed. Provisioning a mailbox
first does **not** help (the per-DirectAdmin-user SpamAssassin hypothesis is
disproven). The write resources are implemented and correct against the spec,
but their acceptance tests are **skipped** (`skipSpamWriteKnownLimitation`)
until the API is fixed; the spam-list DELETE path-encoding stays unverified
as a result. TODO: file an MXroute bug report for the spam-write 500s.

### Tracking the API spec

The official OpenAPI 3 document is served (unauthenticated) at
`https://api.mxroute.com/openapi.yaml` — the same doc `…/docs` renders; there
is **no** `openapi.json` (it 404s). A committed snapshot lives at
[`api/openapi.yaml`](../api/openapi.yaml), the source of truth `API-MAPPING.md`
and the provider schemas are built against.

**Check the snapshot against the official spec** at the start of API-facing
work (auditing schemas, adding/refining a resource, an `info.version` bump
rumor) by **diffing the two files** — never by comparing `info.version`
strings. Upstream can change a path, request body, or `required` array without
bumping the version, so a matching version number does not prove the specs
agree; only a byte-level diff does:

```sh
curl -sS https://api.mxroute.com/openapi.yaml -o /tmp/mxroute-openapi.yaml
diff -u api/openapi.yaml /tmp/mxroute-openapi.yaml
```

No output → in sync. Any diff → the API changed: update the affected
resources/schemas and `API-MAPPING.md`, then replace `api/openapi.yaml` with
the fetched copy and commit it **in the same change** as the code it drives.
See [`api/README.md`](../api/README.md).

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
- **Docs:** `tfplugindocs` via `go generate`, kept current. `make generate`
  needs a **real** terraform on `PATH` — the docker-wrapped `terraform` fails
  its `terraform fmt -recursive ../examples/` step with `No file or directory
  at ../examples` (the wrapper only mounts the tool's cwd). Run it with the
  cached binary, e.g. `PATH="$HOME/.cache/tf-acc:$PATH" make generate`. (Same
  wrapper caveat as `make testacc`; see [TESTS.md](TESTS.md).)

## Merge policy & versioning

- `master` is **PR-only** (server-side ruleset + the local `no-commit-to-branch`
  hook). Required status checks: **`Build`** (build + golangci-lint) and
  **`generate`** (tfplugindocs docs are current). `Acceptance Tests` and
  `Vulnerability scan` run and report but are not required (they skip / can
  fail on a newly-disclosed CVE without a code change). Merge methods: squash
  or merge; 0 required reviewers (solo).
- **`auto-merge: enabled`.** The server-side ruleset (PR-only `master`,
  required `Build` + `generate` checks) makes a manual merge gate redundant,
  so invoking the **push-pr** skill is consent through merge on green CI — the
  agent merges without a separate prompt. The merge still obeys the ruleset
  (required checks, allowed methods); the opt-in skips only the human prompt.
  Closing a PR always needs explicit instruction. The sentinel is read from
  the **default branch**, so the PR that adds it still merges manually —
  auto-merge applies from the next PR.
- **`merge-finalization: enforce`.** Opts into the local
  `merge-finalization.py` hook's hard block: a `gh pr merge` / **push-pr**
  merge is **rejected** while the working tree's `TODO.md` still carries
  completed `- [x]` items (i.e. the merge-time prune — push-pr Step 4.5 — was
  skipped). It backstops the always-on finalization reminder so completed
  items are pruned (to the changelog / git history) at merge rather than left
  as a done-work archive on `master` — the slip that let two `[x]` items
  linger after #52. The hook reads the **working tree**, so a PR that has run
  its finalization (no `[x]` items left) merges cleanly, including the one
  that adds this sentinel.
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
(currently **`1.0.0`** → API major **1**; read it from the committed
`api/openapi.yaml`, or the live `https://api.mxroute.com/openapi.yaml`, field
`info.version` — but detect *spec change* by diffing, not by this string; see
*Tracking the API spec* above).

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
  `1.x`. The jump is **gated on clearing the open live-verification and
  acceptance work in [`TODO.md`](../TODO.md)**: `v1` is the compatibility
  promise, and it can't be made while live-API shapes and acceptance findings
  are still open. Emptying the TODO enables the jump; it does not force it —
  the `0 → 1` call stays deliberate.
- **Every release documents its targeted API version** — a one-line
  `Compatibility: targets MXroute API 1.x` in the release notes / changelog —
  so the tag ↔ API relationship stays explicit even while alpha encodes only
  the minor.

Tags are **annotated**, cut at the merge commit on `master`, via the
`release-tag` skill. See [RELEASING.md](../RELEASING.md) for the release
mechanics.
