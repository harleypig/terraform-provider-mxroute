# terraform-provider-mxroute — Agent Guide

Auto-loaded entry point for AI agents working in terraform-provider-mxroute —
an open-source (MIT) Terraform provider for MXroute email hosting, built on
`terraform-plugin-framework`. Repo conventions and the test layout are pulled
in via the imports at the bottom.

## The few things to internalize first

- **This is a publish-intent OSS provider.** Code quality, docs, and the
  Registry contract matter; `master` is PR-only.
- **The language is Go.** The global `go.md` / `golangci-lint.md` apply;
  provider-dev depth is the `terraform-provider-patterns` skill.
- **The toolchain is pinned in `go.mod`** (`go1.25.8`). A local Go **≥ 1.21**
  is enough — `GOTOOLCHAIN=auto` fetches the pinned toolchain; do **not**
  require a system-Go bump.
- **Acceptance tests hit a LIVE MXroute account** (`TF_ACC`), via the harleydev
  `bin/set_env` credentials — **never** in the default CI gate (see
  [TESTS.md](TESTS.md)).
- **Secrets are write-only.** A create-only value (a mailbox password) uses a
  framework `WriteOnly` attribute + a `*_wo_version` trigger — never a plain
  `Sensitive` field (see [CONVENTIONS.md](CONVENTIONS.md)).
- **The API is `api.mxroute.com`** — flat REST, a `{success,data}` envelope,
  three `X-Server` / `X-Username` / `X-API-Key` headers.

## Where the rest lives

Repo conventions in [CONVENTIONS.md](CONVENTIONS.md); test layout in
[TESTS.md](TESTS.md); dev basics in [../README.md](../README.md). Generic agent
behavior — git/gh workflow, code style, Go tooling, the QA dimensions — comes
from the maintainer's global `~/.claude/` config, which this repo defers to
except where these files override it.

@CONVENTIONS.md
@TESTS.md
