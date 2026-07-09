# TODO

## Acceptance testing

- [ ] Grow `TF_ACC` acceptance coverage further only as needs arise —
  element-content assertions on the remaining list data sources, richer
  multi-attribute update permutations. Not a v1 gate (depth, not correctness).
- [ ] Have harleydev's `bin/mxroute-provider-testacc` optionally set
  `MXROUTE_TEST_UNVERIFIED_DOMAIN` and pass a `-run` filter, so the full v1
  verification (including `TestAccDomainResource_unverified422`) runs through
  the sanctioned runner instead of a hand-exported env var + a direct
  `go test -run`.

## Release

- [ ] Cut `v1.0.0` — the acceptance / live-verification gate is cleared, so
  the deliberate `0 → 1` jump is enabled (CONVENTIONS *Versioning & tagging*):
  the provider adopts the API's current major (1) as its own; the first stable
  tag targets API `1.x`. Cut with the `release-tag` skill; release notes carry
  `Compatibility: targets MXroute API 1.x`. **Breaking since 0.4.0:**
  `email_account.limit` is now read-only.
