# TODO

## Provider Setup

- [ ] Verify against the live account (via acceptance tests) the caveats the
  fan-out flagged in code comments: `/quota` + `/quota/email` enveloping
  (may be unwrapped), and the spam **blacklist** GET response shape (assumed
  `[]string` like the whitelist). Reseller user/package are unverifiable
  without a reseller account.
- [ ] Confirm the documented `ssl_enabled` behavior against the live account:
  the attribute description states it is `false` immediately after domain
  create and flips to `true` asynchronously once AutoSSL issues the cert
  (inferred from DirectAdmin, not the MXroute API). Verify the actual timing
  and whether a post-create refresh is needed.

## Repo Setup

- [ ] Register the provider on the Terraform Registry (see
  [RELEASING.md](RELEASING.md)): sign in, **Publish → Provider →** add
  `harleypig/terraform-provider-mxroute`, upload the GPG public key. The
  existing `v0.1.0` GitHub release ingests automatically once registered
  (currently 404 on registry.terraform.io).
