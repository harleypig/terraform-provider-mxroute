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
- [ ] **Spec-audit refinements (audit done 2026-07-06).** Diffed every
  resource/data source against `api/openapi.yaml` (19 units, each finding
  adversarially verified): 17 units clean, **6 confirmed refinements** on two
  resources. The `email_account password_wo` fix that seeded this is the
  model. Method: per write body, compared the spec's `required:` arrays (POST
  vs PATCH) and `default`/`maximum`/`minLength` bounds against each Go schema
  and request struct. Endpoint coverage is already complete (26 paths / 43
  ops, all implemented — verified 2026-07-06); this is about attribute
  *refinements*, not missing endpoints. Fix items, most severe first:
  - [ ] **medium — `mxroute_email_account` `limit` not create-settable.** The
    schema marks `limit` Optional+Computed, but `createEmailAccountRequest`
    (email_account_resource.go:54-58) omits it and `Create()` (lines 193-197)
    never sends it — only PATCH does, so a `limit` set at create is silently
    dropped. Add a `Limit *int64` field (json tag `limit,omitempty`) to the
    create struct and set `Limit: int64PtrFromValue(plan.Limit)` in
    `Create()`, mirroring `quota`; the read-back already sets `Limit`. Spec:
    POST body `limit` default/maximum 9600 (openapi.yaml:690-694).
  - [ ] **medium — `mxroute_reseller_user` `password_wo` wrongly `Required`.**
    PATCH doesn't require a password, so mirror the email_account fix in FULL:
    flip line 103-104 from `Required: true` to `Optional: true` (keep
    `WriteOnly`); add the create-time null/empty guard in `Create()` (per
    email_account_resource.go:180-188) **and** the rotation guard in
    `Update()`'s `PasswordWOVersion`-change block (per :280-288) so a version
    bump can't PATCH an empty password; update the MarkdownDescription to
    "required only on create." Add create-without-password and
    update-with-omitted regression tests. Spec: POST
    `required:[username,email,password,package]`, PATCH has no `required:`
    (openapi.yaml:1187, 1238-1245). (Acceptance-testing needs a reseller
    account — the fix is spec-provable regardless.)
  - [ ] **low — `mxroute_email_account` `limit` upper-bound validator.** Add
    `int64validator.AtMost(9600)` to the `limit` attribute
    (email_account_resource.go:107-114) so an out-of-range value fails at plan
    time. Spec: POST `limit` `maximum: 9600` (openapi.yaml:694).
  - [ ] **low — `password_wo` missing `minLength(8)` validator (both
    resources).** Add `stringvalidator.LengthAtLeast(8)` to `password_wo` on
    `mxroute_email_account` (email_account_resource.go:90-94) and
    `mxroute_reseller_user` (reseller_user_resource.go:101-105); leave the
    upper/lower/number complexity rule to the API (exact regex unpublished).
    Spec: `password` `minLength: 8` on both bodies (openapi.yaml:685/741,
    1198/1245).
  - [ ] **low — `mxroute_reseller_user` `username` bounds (needs live-API
    confirmation).** Add `stringvalidator.LengthBetween(1, 10)` +
    `RegexMatches(^[a-z0-9_]+$)` (reseller_user_resource.go:83-89). Deferred:
    the constraint is prose-only in the spec `description` ("1-10 chars,
    lowercase letters, numbers, underscores") with no
    `minLength`/`maxLength`/`pattern` keyword — confirm the exact bounds live
    before enforcing (openapi.yaml:1191).

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

## Provider comparison backlog (vs demon-tf-provider-mxroute)

Surfaced by the `compare-mxroute-providers` workflow comparing this provider
against a more mature existing one, `demon-tf-provider-mxroute` (the full
per-module analysis + file:line pointers is in the local, **untracked**
`FINDINGS.md`). Verdict: **keep this provider, cherry-pick these** — ours holds
the correctness/security edge (write-only passwords, the real
`{success,data,error}` envelope, 429-only retry, idempotent deletes, an
httptest seam), so demon's wins are structural/ergonomic, not a reason to swap.

### Ergonomics & DRY

The DRY pass itself is done (merged); these decisions from it are kept — not
pending work:

- **Declined: a domain/email format-validator library.** demon ships
  `DomainName` / `Email` format validators, but this provider **deliberately
  defers FORMAT validation to the API** — verified by the spec audit's rejected
  `email` finding (its validators are enums / ranges / presence only, never
  `format:`). Adding format regexes would fight that convention and risk
  rejecting valid inputs. The **enum** case is covered (OneOf, above); the
  spec-grounded **range** validators (`password` minLength, `limit` AtMost,
  `username` bounds) are the separate audit items under *Provider Setup*.
- **Deferred (YAGNI): extracting the client into `internal/client`.** `client.go`
  is already a self-contained `Client` with a thin `Do` and an httptest seam
  (`client_test.go`); a separate package buys only export churn at this size.
  Revisit only if typed per-endpoint methods become worthwhile.

### Data-source coverage

- [ ] Add the six data sources demon has and we lack, modeled on the existing
  `email_accounts_data_source` (typed struct + `ListValueFrom`, keep our `id`
  convention), each with a docs page + example — all are thin read-only wrappers
  over reads the client already performs: singular `mxroute_reseller_package`
  and `mxroute_reseller_user`; and list `mxroute_pointers`, `mxroute_forwarders`,
  `mxroute_spam_blacklist`, `mxroute_spam_whitelist`.

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
- [ ] Whether the API requires `@`/`+` percent-encoded in path segments (e.g. a
  spam entry or forwarder alias with `+`). `pathSeg` uses `url.PathEscape`,
  which encodes `*` and `/ # ? space` but leaves `@`/`+` as RFC-valid pchar. If
  a live DELETE of an entry containing `@`/`+` misses, switch `pathSeg` to a
  stricter encoder (encode those too). Exercise with a `foo+bar@x` alias / entry
  against the test domain.
