# 3. Keep the API client in-package (no internal/client split)

- Status: accepted
- Date: 2026-07-06

## Context

`demon-tf-provider-mxroute` places its API client in a dedicated
`internal/client` package with thin typed per-endpoint methods. Extracting
ours the same way was considered during the ergonomics/DRY pass, for cleaner
layering and isolated testing.

Our client (`internal/provider/client.go`) is already a self-contained `Client`
with a single thin `Do` method: resources marshal their own request/response
types and call `Do`, so adding a resource never edits the client. It is
independently unit-tested with an `httptest` seam (`client_test.go`).

## Decision

**Keep the client in the `provider` package.** Do not split it into
`internal/client`, and do not add typed per-endpoint methods, at the current
size.

## Consequences

- Avoids exporting `Client`, `APIError`, `NewClient`, `IsNotFound`, etc. purely
  to cross a package boundary — export churn that buys nothing today.
- The httptest-based unit tests already give the isolated testing a separate
  package would provide.
- Revisit if typed per-endpoint methods (centralized path construction) become
  worthwhile as the provider grows — a YAGNI line, not a permanent bar.
