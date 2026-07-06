# TODO

## Provider Setup

- [ ] Verify against the live account (via acceptance tests) the caveats the
  fan-out flagged in code comments: `/quota` + `/quota/email` enveloping
  (may be unwrapped), and the spam **blacklist** GET response shape (assumed
  `[]string` like the whitelist). Reseller user/package are unverifiable
  without a reseller account. The demon provider decodes `/quota`
  **unenveloped**, corroborating that read (its `/quota` endpoints were also
  500ing upstream at comparison time).
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

## Provider comparison backlog (vs demon-tf-provider-mxroute)

Surfaced by the `compare-mxroute-providers` workflow comparing this provider
against a more mature existing one, `demon-tf-provider-mxroute` (the full
per-module analysis + file:line pointers is in the local, **untracked**
`FINDINGS.md`). Verdict: **keep this provider, cherry-pick these** — ours holds
the correctness/security edge (write-only passwords, the real
`{success,data,error}` envelope, 429-only retry, idempotent deletes, an
httptest seam), so demon's wins are structural/ergonomic, not a reason to swap.

### Correctness fixes (present in the released v0.2.0)

- [ ] `url.PathEscape` the spam blacklist/whitelist entry DELETE paths — entries
  are emails/wildcards (`*@spammer.test`) with `@`/`*`/`+` that must be
  percent-encoded, but the path is concatenated raw → broken deletes. Audit
  sibling resources for the same raw-path concatenation; add a regression test.
- [ ] Fix the `catch_all` empty-string validation hole — validation keys only on
  `IsNull()`, so `type = address` with `address = ""` slips through and PATCHes
  an empty address (and `type = fail`/`blackhole` with `""` wrongly errors).
  Reject `""` for `address`, ignore it otherwise; regression test.
- [ ] Change `mxroute_forwarder.destinations` from `List` to `Set` (with
  `setvalidator.SizeAtLeast(1)`): with `List` + `RequiresReplace`, the API
  reordering destinations forces a destroy/recreate of a live forwarder. Also
  affects harleydev's fan-out forwarders (`support: [a, b]`).

### Ergonomics & DRY

- [ ] Add `internal/providerutil` with `ResourceClient`/`DataSourceClient`
  Configure helpers and convert all 20 Configure sites — collapses a duplicated
  17-line `*Client` type-assertion block (×20, ~340 lines) to ~2 lines each,
  ~300 lines net (a Rule-of-Three violation), with the error wording defined
  once. Doable without the subpackage split.
- [ ] Add a shared validators library (`DomainName` / `LocalPart` / `Email` /
  `NumericOrUnlimited`) and apply it across every resource's domain / username /
  email / quota attributes — ours has essentially no plan-time input validation
  (only two bespoke `catch_all` validators), so users hit apply-time API 422s.
- [ ] Extract the API client into its own `internal/client` package over the
  existing `Do` — cleaner layering, isolated httptest-based testing. Optionally
  add thin typed per-endpoint methods incrementally (demon's real structural
  win: centralizes path construction) — gate against YAGNI for the current size.
- [ ] Smaller DRY nits: `stringvalidator.OneOf` to replace the hand-rolled
  `catchAllTypeValidator`; a shared `apply()` helper for Create/Update in the
  singleton resources (`catch_all`, `spam_settings`); exponential-backoff-with-
  cap as the no-header fallback for 429 retries (do **not** copy demon's 5xx
  retry — it retries non-idempotent POST/PATCH).

### Data-source coverage

- [ ] Add the six data sources demon has and we lack, modeled on the existing
  `email_accounts_data_source` (typed struct + `ListValueFrom`, keep our `id`
  convention), each with a docs page + example — all are thin read-only wrappers
  over reads the client already performs: singular `mxroute_reseller_package`
  and `mxroute_reseller_user`; and list `mxroute_pointers`, `mxroute_forwarders`,
  `mxroute_spam_blacklist`, `mxroute_spam_whitelist`.

### Endpoint coverage (what BOTH providers miss)

- [ ] Cross-check the **live** OpenAPI spec
  (`https://api.mxroute.com/openapi.yaml`, ~32 paths / ~71 operations) against
  what **both** providers implement. Demon was built from a **stale**
  `spec.json` (26 paths / 43 ops), and the comparison only surfaced what each
  provider has that the *other* lacks — so any endpoint absent from **both**
  went unnoticed. Diff the live spec's operations against our registered
  resources/data sources (and demon's) to find **unimplemented** MXroute
  capabilities worth adding as new resources/data sources. Distinct from the
  "audit each resource against the spec" item above (that refines what we
  *have*; this finds what *neither* provider has). Pull the live spec with the
  three `MXROUTE_*` creds (it is auth-gated: `curl -H X-Server/X-Username/
  X-API-Key https://api.mxroute.com/openapi.yaml`).

### CI & governance

- [ ] Resolve the CI credential gap: decide and document whether to wire
  `MXROUTE_SERVER/USERNAME/APIKEY` as repo secrets for the acceptance job (it
  currently sets `TF_ACC=1` with none, so the live path never runs) or that
  live acceptance is intentionally not CI-run; add a credential-free client
  unit-test job. (Ties into the throwaway-test-domain item above.)
- [ ] Fix the stale `.github/CODEOWNERS` (still the scaffold's
  `* @hashicorp/terraform-core-plugins`) and add `CONTRIBUTING.md` +
  `SECURITY.md` (near-verbatim from demon, swapping URLs).
- [ ] Add Example Usage / `examples/` for the data sources that lack them
  (`mxroute_quota`, `mxroute_verification_key`, `mxroute_email_quota`).

### Live-API investigations (via acceptance tests)

- [ ] Whether the `email_account` CREATE body accepts `limit` — ours omits it,
  risking a provider-inconsistent-result error if a user sets `limit` at create.
- [ ] Whether the reseller API accepts a per-user quota PATCH — if not, ours'
  settable `mxroute_reseller_user` quota input is a misleading no-op and should
  become computed (as demon models it).
