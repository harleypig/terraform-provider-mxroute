# TODO

## Acceptance testing

- [x] Added a `TESTARGS` passthrough to the `testacc` make target for scoped
  live runs (`make testacc TESTARGS='-run TestAccFoo'`).
- [x] **v1 live-verification passed.** The suite is green/skip against the
  live account: `TestAccDomainResource_unverified422`,
  `TestAccForwarderResource_sentinelDestination`, `TestAccQuotaDataSource`,
  and the full resource/data-source lifecycle all confirmed. The two live
  findings became **documented limitations** (CONVENTIONS *Known
  limitation*), not blockers: spam writes 500 (a mailbox does not help; write
  tests skipped via `skipSpamWriteKnownLimitation`, MXroute ticket deferred)
  and `email_account.limit` is unreliable (made **read-only**, MXroute ticket
  deferred). The v0→v1 gate is clear.
- [ ] Grow `TF_ACC` acceptance coverage further only as needs arise —
  element-content assertions on the remaining list data sources, richer
  multi-attribute update permutations. Not a v1 gate (depth, not correctness).

## Release

- [ ] Cut `v1.0.0` — the acceptance / live-verification gate is cleared, so
  the deliberate `0 → 1` jump is enabled (CONVENTIONS *Versioning & tagging*):
  the provider adopts the API's current major (1) as its own; the first stable
  tag targets API `1.x`. Cut with the `release-tag` skill; release notes carry
  `Compatibility: targets MXroute API 1.x`. **Breaking since 0.4.0:**
  `email_account.limit` is now read-only.
