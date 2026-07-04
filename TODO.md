# TODO

## Provider Setup

- [ ] Verify against the live account (via acceptance tests) the caveats the
  fan-out flagged in code comments: `/quota` + `/quota/email` enveloping
  (may be unwrapped), and the spam **blacklist** GET response shape (assumed
  `[]string` like the whitelist). Reseller user/package are unverifiable
  without a reseller account.

## Repo Setup

- [x] Release signing + `v0.1.0`: GPG key, `GPG_PRIVATE_KEY`/`PASSPHRASE`
  secrets, and the tag — the release workflow built and GPG-signed all
  platforms and published the GitHub release (`v0.1.0`).
- [ ] Register the provider on the Terraform Registry (see
  [RELEASING.md](RELEASING.md)): sign in, **Publish → Provider →** add
  `harleypig/terraform-provider-mxroute`, upload the GPG public key. The
  existing `v0.1.0` GitHub release ingests automatically once registered
  (currently 404 on registry.terraform.io).
