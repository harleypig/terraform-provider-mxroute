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
- [ ] Audit every resource/data source against the MXroute OpenAPI spec
  (`https://api.mxroute.com/openapi.yaml`) for refinements like the
  `email_account` `password_wo` fix. That one came from diffing the spec's
  create body (`POST … required: [username, password]`) against the update body
  (`PATCH … required: []`): an attribute the schema marks `Required` that the
  API only requires on **create** should be `Optional` + a create-time
  validator, not `Required`. Sweep for: (a) each write body's `required` array
  vs the schema's `Required`/`Optional`; (b) `Optional`+`Computed`+default
  handling vs the API's documented create-time defaults; (c) `RequiresReplace`
  attributes vs what `PATCH` actually accepts in place. Record each mismatch as
  its own fix.

## Repo Setup

- [ ] Set up a verified throwaway test domain (e.g. `throwaway.harleypig.dev`)
  on the MXroute account so the acceptance tests can run — they manage a
  `mxroute_domain` resource, and MXroute rejects adding any new domain (HTTP
  422 `Domain verification required`) until a DNS TXT ownership record proves
  it, so a fresh throwaway can't be stood up in-test. Steps: add the subdomain
  in the MXroute panel/API, publish the required TXT record (its DNS lives in
  Linode via harleydev's `domains/` config), complete verification, then set
  `MXROUTE_TEST_DOMAIN` to it — locally (for `make testacc`) and as a CI secret
  (the `Acceptance Tests` job currently **skips** because it's unset). The
  `testAccTestDomain` guard forbids `harleypig.com`, so a dedicated throwaway is
  the only way to exercise the domain-managing acceptance tests. (The
  `email_account` password change was verified live via a dev-override against
  the existing `harleypig.com` domain with a throwaway mailbox, sidestepping
  this — but the full suite needs the verified domain.)
- [ ] Register the provider on the Terraform Registry (see
  [RELEASING.md](RELEASING.md)): sign in, **Publish → Provider →** add
  `harleypig/terraform-provider-mxroute`, upload the GPG public key. The
  existing `v0.1.0` GitHub release ingests automatically once registered
  (currently 404 on registry.terraform.io).
