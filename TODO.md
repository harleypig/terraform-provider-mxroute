# TODO

## Acceptance testing

- [x] **RESOLVED (2026-07-08) — the test domain creates again.** The
  `HTTP 422 "Domain verification required"` block on `harleypig.dev` was an
  **MXroute-side glitch** (their Tailscale DNS acting up), cleared by MXroute
  tier-3 support — **not** a per-domain cooldown/block, and **not** anything on
  our side: our `_da-verify` TXT was published + authoritative throughout, and
  `/verification-key` matched byte-for-byte. A re-run then created the domain
  and drove the full lifecycle (coverage 24.9%→51.5%). **If it recurs:** a
  domain-add 422 while `dig TXT _da-verify-<key>.<domain>` resolves
  authoritatively **and** `TestAccVerificationKeyDataSource` passes is
  MXroute-side — escalate, don't touch DNS/DNSSEC. Full detection signature +
  incident writeup live in harleydev `e2e/mxroute.md` ("Known incident");
  harleydev's `bin/mxroute-provider-testacc` now DNS-preflights before the run.
- [ ] Grow `TF_ACC` acceptance coverage toward all resources and data sources
  (CRUD + import round-trips), scoped to **provider-internals the fabric can't
  surface** — `ImportState`, write-only `password_wo` create/rotate, error
  paths, and the read-only data sources. Complements harleydev's e2e tier
  (`e2e/mxroute.md`), which exercises the *applied* CRUD path via the
  mxroute-foundation-fabric modules. Run the suite with
  `bin/mxroute-provider-testacc` (see [TESTS.md](.claude/TESTS.md)).
  A coverage audit filled the known gaps: catch-all's `address`↔`type`
  config-validator error paths (incl. the empty-string address) and its `type`
  `OneOf`, the `spam_settings.high_score` range validator (all plan-time, so
  they run in the default CI gate), and `email_account` password **rotation**
  (`TestAccEmailAccountResource_passwordRotation`, which creates the domain so
  it waits on the block above). Every resource already has `ImportState` +
  `CheckDestroy` and every in-place-updatable resource an update step. What is
  left is depth, not breadth — richer data-source content assertions and
  multi-attribute update permutations, added as needs arise.
- [ ] Confirm the live assertions now baked into the suite pass on a green
  `make testacc` run (the 422 block is **resolved** 2026-07-08, so these run
  now; a *fully* green run still waits on the spam-writes-500 bug below).
  `TestAccPointerResource` asserts the
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
  (both spam data sources pass, so the GET shapes are fine). **Confirmed still
  reproducing on the 2026-07-08 live run** once the 422 cleared — the three
  spam writes are the only failures; reads, validators, and
  `VerificationKeyDataSource` pass. Investigate fresh-vs-established domain (a
  read against harleypig.com is safe anytime); open an MXroute ticket if it
  reproduces generally. Blocks the spam-entry DELETE path (and its `@`/`+`
  encoding) until the creates succeed.
