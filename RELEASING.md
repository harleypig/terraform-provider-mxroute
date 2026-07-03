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

Releases are **semver** (see the global `git.md`): `v0.y.z` is alpha (breakage
expected, loose `y.z`); the `0 → 1` jump declares stability. **Do not tag
before there is releasable code** — a published Registry version cannot be
unpublished.

1. Merge the work to `master`; ensure CI is green.
2. Cut an **annotated** tag at the merge commit and push it (use the
   `release-tag` skill, which automates this):

   ```sh
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
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
