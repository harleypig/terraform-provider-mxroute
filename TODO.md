# TODO

## Acceptance testing

- [ ] **BLOCKED (MXroute-side) — the test domain can't be created.** Adding
  `harleypig.dev` fails `HTTP 422 BUSINESS_ERROR "Domain verification
  required"` via **both** the API and the panel, even though the required
  `_da-verify-7363530376cb3483f57f76d42a9055eb` TXT (value `domain-verified`)
  is published, authoritative (served by `ns4.linode.com`), and matches the
  panel and the API's own `/verification-key` byte-for-byte. Every test that
  creates the throwaway domain dies at step 1; account-level data-source tests
  still pass, so auth/client are fine. Most likely a per-domain block or
  cooldown from the 2026-07-09 rapid add/remove cycling — which **disproves**
  that run's "the 422 is moot in practice / TXT standing → re-adds always
  pass" note. MXroute support ticket opened 2026-07-08 (asks: cooldown/block?
  account-wide broken check? a test/sandbox path or dry-run flag to avoid the
  churn — none is documented in the spec or docs). Re-run
  `bin/mxroute-provider-testacc` once MXroute clears it.
- [ ] Grow `TF_ACC` acceptance coverage toward all resources and data sources
  (CRUD + import round-trips), scoped to **provider-internals the fabric can't
  surface** — `ImportState`, write-only `password_wo` create/rotate, error
  paths, and the read-only data sources. Complements harleydev's e2e tier
  (`e2e/mxroute.md`), which exercises the *applied* CRUD path via the
  mxroute-foundation-fabric modules. Run the suite with
  `bin/mxroute-provider-testacc` (see [TESTS.md](.claude/TESTS.md)).
- [ ] Confirm the live assertions now baked into the suite pass on a green
  `make testacc` run (**blocked by the domain-verification item above** —
  these all create the test domain). `TestAccPointerResource` asserts the
  domain's `pointers` list holds the created pointer (the `Domain.pointers`
  decode against a live populated response);
  `TestAccForwarderResource_plusInAlias` exercises the `+`-in-alias path
  encoding via `CheckDestroy`; `TestAccEmailAccountResource` sets `limit` at
  create. If the pointers assertion fails, the object's names live in the
  values not the keys — adjust `decodePointerNames`. If the `+` test's
  `CheckDestroy` fails, make `pathSeg` escape `@`/`+` too.

## Features & fixes

- [ ] **Bug: spam writes 500 on a fresh domain.** `mxroute_spam_settings`,
  `mxroute_spam_blacklist_entry`, and `mxroute_spam_whitelist_entry` all fail
  `HTTP 500 Failed to update spam settings/list` against a just-created domain
  (both spam data sources pass, so the GET shapes are fine). Investigate
  fresh-vs-established domain (a read against harleypig.com is safe anytime);
  open an MXroute ticket if it reproduces generally. Blocks the spam-entry
  DELETE path (and its `@`/`+` encoding) until the creates succeed.
