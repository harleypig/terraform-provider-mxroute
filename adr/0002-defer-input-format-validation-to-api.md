# 2. Defer input format validation to the API

- Status: accepted
- Date: 2026-07-06

## Context

`demon-tf-provider-mxroute` ships a shared format-validator library
(`DomainName`, `Email`, …) applied to domain/email inputs, and adopting one was
considered for this provider. Separately, a spec audit of every resource
against `api/openapi.yaml` looked for missing plan-time validation.

That audit's own finding on `reseller_user.email` was **rejected** on the
grounds that this provider consistently defers format checks to the API: its
validators are enums (`catch_all.type`), int64 ranges (`spam_settings`,
`email_account.limit`), string-length bounds (`password_wo`), and cross-field
presence rules (`catch_all` address) — never a `format:` regex. The comparable
user-supplied `catch_all.address` likewise has no format validator.

## Decision

**Do not add a domain/email format-validator library. Keep deferring input
*format* validation to the API.** Plan-time validators are added only for
machine-checkable, spec-grounded constraints — enums, numeric ranges, and
length bounds — not format regexes.

## Consequences

- A malformed domain/email surfaces as a clear API error at apply time rather
  than a plan-time regex rejection. This avoids the larger risk: a too-strict
  regex rejecting inputs the API would actually accept.
- The **enum** case is covered by `stringvalidator.OneOf`; spec-grounded
  **range/length** validators (`limit` AtMost 9600, `password_wo` minLength 8)
  are implemented. A prose-only `username` bound remains pending live
  confirmation (see `TODO.md`).
- Revisit only if the API's format rules become documented and stable enough
  that a plan-time check would strictly help without false rejections.
