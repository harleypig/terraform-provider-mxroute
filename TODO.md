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
- [ ] **Spec-audit refinements — 1 of 6 remaining.** A spec audit (2026-07-06)
  against `api/openapi.yaml` confirmed 6 attribute refinements on two resources
  (17 of 19 units were clean). Five are fixed and merged — email_account
  `limit`-on-create plus `limit`/`password_wo` plan validators, and
  reseller_user `password_wo` Optional with create/rotation guards and a
  `minLength` validator. The last is deferred pending live confirmation:
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

- [x] Add the six data sources demon has and we lack — done: singular
  `mxroute_reseller_package` / `mxroute_reseller_user`, and list
  `mxroute_forwarders` / `mxroute_pointers` / `mxroute_spam_blacklist` /
  `mxroute_spam_whitelist`, each with an example, generated docs, and an
  acceptance test (the domain-scoped ones skip without the test domain; the
  reseller singular ones require reseller access).

### CI & governance

- [ ] Resolve the CI credential gap: decide and document whether to wire
  `MXROUTE_SERVER/USERNAME/APIKEY` as repo secrets for the acceptance job (it
  currently sets `TF_ACC=1` with none, so the live path never runs) or that
  live acceptance is intentionally not CI-run; add a credential-free client
  unit-test job. (Ties into the throwaway-test-domain item above.)
- [ ] Fix the stale `.github/CODEOWNERS` (still the scaffold's
  `* @hashicorp/terraform-core-plugins`) and add `CONTRIBUTING.md` +
  `SECURITY.md` (near-verbatim from demon, swapping URLs).
- [ ] Ensure **every** resource and data source has a registry example so its
  "Example Usage" (and, for importable resources, "Import") section renders on
  the registry.terraform.io page. tfplugindocs emits those sections only from
  `examples/resources/<name>/resource.tf`,
  `examples/data-sources/<name>/data-source.tf`, and
  `examples/resources/<name>/import.sh`. Audit the full resource/data-source set
  against the `examples/` tree on every addition. Current gap (2026-07-06): all
  resources are covered, but three data sources lack an example `mxroute_quota`,
  `mxroute_email_quota`, `mxroute_verification_key` -- add a `data-source.tf`
  for each and regenerate docs.
- [ ] Flesh out the provider **Overview** on the registry.terraform.io landing
  page. It renders `docs/index.md`, generated from `templates/index.md.tmpl`
  (plus the provider schema) — currently only a two-sentence blurb before
  Requirements/Example Usage. Expand the template's prose into a proper
  overview: what the provider manages (the resource/data-source catalog by
  area — domains, mailboxes, forwarders/pointers, catch-all, spam, reseller),
  the auth model (three headers with `MXROUTE_*` env-var fallback), the
  write-only password handling, and links to the MXroute API/docs. Then
  regenerate `docs/index.md` (`make generate`) and confirm it via CI `generate`.

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
