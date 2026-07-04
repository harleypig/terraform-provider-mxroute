# terraform-provider-mxroute

A [Terraform](https://www.terraform.io) provider for
[MXroute](https://mxroute.com) email hosting, built on the
[terraform-plugin-framework][fw]. It manages MXroute domains, mailboxes,
forwarders, pointers, and spam settings through the MXroute REST API
(`api.mxroute.com`).

> **Status: early development.** The scaffold is in place; resources are being
> added. Not yet published to the Terraform Registry.

## Why

MXroute shipped a public REST API (v1.0.0) but has no Terraform provider. This
brings MXroute account management under Terraform, so email hosting is managed
as code alongside the rest of the infrastructure.

## Requirements

- **Terraform >= 1.11.** The `mxroute_email_account` and `mxroute_reseller_user`
  resources use [write-only arguments][wo], which Terraform added in 1.11. The
  provider has no other version-gated features, so 1.11 is the baseline for the
  whole provider.

[wo]: https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments

## Development

A local `go` **≥ 1.21** is enough — `go.mod` pins the toolchain and
`GOTOOLCHAIN=auto` fetches it. Terraform **≥ 1.11** (write-only arguments are
exercised by the docs generation and acceptance tests; see *Requirements*).

```sh
go build ./...     # build
go test ./...      # unit tests
make testacc       # acceptance tests (TF_ACC=1; hits a real MXroute account)
```

Acceptance tests need MXroute API credentials in the environment
(`X-Server` / `X-Username` / `X-API-Key`).

See [API-MAPPING.md](API-MAPPING.md) for how each MXroute API endpoint maps to
the provider's resources and data sources.

## License

[MIT](LICENSE).

[fw]: https://github.com/hashicorp/terraform-plugin-framework
