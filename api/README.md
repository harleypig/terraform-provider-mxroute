# MXroute API spec snapshot

[`openapi.yaml`](openapi.yaml) is a **committed snapshot** of the official
MXroute OpenAPI 3 document, the source of truth this provider is built
against. It backs [`../API-MAPPING.md`](../API-MAPPING.md) (the endpoint ↔
resource map) and every schema/`required`-array decision in the provider.

Keeping a copy in-repo means the provider's design intent is pinned to a
concrete spec: a change upstream shows up as a **diff against this file**, not
as a silent drift you only discover when a resource breaks.

## The official version

The canonical spec is served, unauthenticated, at:

```text
https://api.mxroute.com/openapi.yaml
```

(`https://api.mxroute.com/docs` renders the same document; there is **no**
`openapi.json` — that path 404s.) The spec declares its own version in
`info.version` (this snapshot: `1.0.0`), but see the check below — **do not
rely on that number to detect change.**

## Checking our copy against the official one

Fetch the live spec and **diff the two files**. Compare *content*, not the
`info.version` string — upstream can edit a path, body, or `required` array
without bumping `info.version`, so a version match does **not** prove the
specs are identical:

```sh
curl -sS https://api.mxroute.com/openapi.yaml -o /tmp/mxroute-openapi.yaml
diff -u api/openapi.yaml /tmp/mxroute-openapi.yaml
```

- **No output** — our snapshot matches the official spec; nothing to do.
- **Any diff** — the upstream API changed. Review the hunks, then:
  1. Update the provider for any behavioral change (a new/changed path, a
     shifted `required` array, a new enum value, …).
  2. Refresh [`../API-MAPPING.md`](../API-MAPPING.md) if endpoints moved.
  3. Replace this snapshot with the fetched file (`cp /tmp/mxroute-openapi.yaml
     api/openapi.yaml`) and commit it **with** the provider change, so the
     snapshot and the code that consumes it move together.

The agent runs this check per the "Tracking the API spec" convention in
[`../.claude/CONVENTIONS.md`](../.claude/CONVENTIONS.md).
