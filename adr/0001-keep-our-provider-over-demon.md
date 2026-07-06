# 1. Keep our provider rather than adopting demon-tf-provider-mxroute

- Status: accepted
- Date: 2026-07-06

## Context

After this provider was built, an existing, more mature MXroute provider was
found — `demon-tf-provider-mxroute`. A workflow (`compare-mxroute-providers`)
compared the two module by module (the full per-module analysis with file:line
pointers is in the local, untracked `FINDINGS.md`). The question was whether to
adopt demon's provider and retire ours, or keep ours and cherry-pick demon's
good ideas.

## Decision

**Keep this provider and cherry-pick from demon.** Ours holds the
correctness/security edge — write-only passwords (never persisted to state),
the real `{success,data,error}` envelope handling, 429-only retry (demon
retries non-idempotent 5xx POST/PATCH), idempotent deletes, and an httptest
seam. Demon's advantages are structural and ergonomic (helpers, data-source
breadth), not correctness — so they are worth importing as individual
improvements, not a reason to switch.

## Consequences

- Demon's structural/ergonomic wins were tracked as backlog items and have
  since been implemented here (the DRY pass, the six added data sources, the
  correctness fixes).
- We do **not** copy demon's 5xx retry (it retries non-idempotent methods).
- `FINDINGS.md` is the local, untracked detail; it is not committed.
