# Contributing

Thanks for your interest in improving the MXroute Terraform provider. This
document covers the local workflow; please also read the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Prerequisites

- **Go** — the version is pinned in `go.mod`; a local Go ≥ 1.21 builds it via
  `GOTOOLCHAIN=auto`, which fetches the pinned toolchain.
- **Terraform ≥ 1.11** — the baseline for the provider (the
  `mxroute_email_account` and `mxroute_reseller_user` resources use write-only
  arguments, added in Terraform 1.11). Terraform is also required to generate
  docs; an older CLI silently drops the write-only annotations.

## Build

```sh
make build      # go build ./...
make install    # go install ./... (into $GOBIN)
```

## Format, lint, and pre-commit

```sh
make fmt        # gofmt -s -w
make lint       # golangci-lint run
```

The repository uses [pre-commit](https://pre-commit.com); the fix config
auto-formats and the check config gates:

```sh
pre-commit run --config .pre-commit-config-fix.yaml --all-files
pre-commit run --all-files
```

## Tests

```sh
make test       # unit tests (credential-free; runs in CI)
make testacc    # acceptance tests — see the caveats below
```

Acceptance tests (`TF_ACC=1`) exercise real Terraform against the provider.
Most **manage live MXroute resources**, so they cost money, mutate a real
account, and are gated by `PreCheck` (they skip without
`MXROUTE_SERVER` / `MXROUTE_USERNAME` / `MXROUTE_API_KEY`) and by
`MXROUTE_TEST_DOMAIN` (a throwaway domain — never a production one). Plan-time
validation tests run credential-free. Live acceptance is intentionally **not**
run in CI.

## Documentation

The Registry docs under `docs/` are generated — never edit them by hand:

```sh
make generate   # tfplugindocs; needs Terraform >= 1.11 on PATH
```

Every resource/data source needs an example under `examples/` (and
importable resources an `import.sh`) for its "Example Usage"/"Import" section to
render. CI fails if `docs/` is out of date, so run `make generate` and commit
the result whenever a schema, example, or template changes.

## Pull requests

- Keep the branch focused; group commits by theme with
  [Conventional Commit](https://www.conventionalcommits.org/) messages.
- Ensure `pre-commit` is clean and the required CI checks (**Build**,
  **generate**) pass.
- Add or update tests for behavior changes, and regenerate docs when the schema
  changes.
