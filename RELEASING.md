# Releasing

How a signed release of `terraform-provider-mxroute` is cut and published to
the [Terraform Registry][registry]. The pipeline (`.goreleaser.yml` +
`.github/workflows/release.yml`) is already wired; this documents the one-time
setup and the per-release steps.

## One-time setup

### 1. GPG signing key

The Registry requires releases to be GPG-signed. Generate a key once:

```sh
gpg --quick-generate-key "Alan Young <you@harleypig.com>" rsa4096 sign 0
gpg --list-secret-keys --keyid-format=long      # note the KEYID
```

### 2. Repo secrets

The release workflow imports the key from two secrets:

```sh
gh secret set GPG_PRIVATE_KEY --repo harleypig/terraform-provider-mxroute \
  < <(gpg --armor --export-secret-key <KEYID>)
gh secret set PASSPHRASE --repo harleypig/terraform-provider-mxroute
```

### 3. Register on the Terraform Registry

- Sign in at <https://registry.terraform.io> with GitHub.
- **Publish → Provider →** add `harleypig/terraform-provider-mxroute`.
- Add the GPG **public** key: `gpg --armor --export <KEYID>`.

## Cutting a release

Releases are **semver with the MAJOR aligned to the MXroute API's major** —
the full policy (and why literal API-lockstep is rejected) is in
[.claude/CONVENTIONS.md](.claude/CONVENTIONS.md) › *Versioning & tagging*. In
short:

- **MAJOR** = the API major you target (API `info.version` is `1.0.0` today →
  major `1`); a breaking API `2.0.0` makes the next release `2.0.0`.
- **MINOR / PATCH** move on the provider's own cadence — a new resource is a
  minor, a fix/deps/docs release is a patch, both **without** an API version
  change.
- **Stable (current):** the provider crossed the deliberate `0 → 1` jump at
  `v1.0.0`, adopting the API major into the tag (major `1` targets API `1.x`).
  `v1` is a compatibility promise, so a breaking change now requires a major
  bump (`git.md`: strict `y.z` once `X ≥ 1`).
- **Note the targeted API version** in the release — a `Compatibility: targets
  MXroute API 1.x` line.

**Do not tag before there is releasable code** — a published Registry version
cannot be unpublished.

1. Merge the work to `master`; ensure CI is green.
2. Cut an **annotated** tag at the merge commit and push it (use the
   `release-tag` skill, which automates this):

   ```sh
   git tag -a v1.0.0 -m "v1.0.0"
   git push origin v1.0.0
   ```

3. The `release` workflow builds all platforms, signs the `SHA256SUMS` with
   your key, and publishes the GitHub release. The Registry ingests the new
   version automatically.

## Artifacts (per `vX.Y.Z` tag)

GoReleaser produces:

- `terraform-provider-mxroute_<ver>_<os>_<arch>.zip`
- `terraform-provider-mxroute_<ver>_manifest.json`
- `terraform-provider-mxroute_<ver>_SHA256SUMS`
- `terraform-provider-mxroute_<ver>_SHA256SUMS.sig` (GPG detached signature)

[registry]: https://registry.terraform.io
