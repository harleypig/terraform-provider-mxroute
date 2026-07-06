# Architecture Decision Records

Short records of significant, deliberate decisions for this provider — the
context, the choice, and why — so a considered "we decided X (and not Y)" isn't
re-litigated or lost. Format is lightweight [MADR][madr]. These are **records
of decisions already made**, not open work; open work lives in
[`../TODO.md`](../TODO.md).

| ADR | Decision |
|-----|----------|
| [0001](0001-keep-our-provider-over-demon.md) | Keep our provider rather than adopting `demon-tf-provider-mxroute` |
| [0002](0002-defer-input-format-validation-to-api.md) | Defer input **format** validation to the API (no format-validator library) |
| [0003](0003-keep-api-client-in-package.md) | Keep the API client in-package (no `internal/client` split) |

[madr]: https://adr.github.io/madr/
