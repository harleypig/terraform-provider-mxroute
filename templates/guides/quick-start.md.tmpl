---
page_title: "Quick Start"
description: |-
  Stand up an MXroute domain with a mailbox and a forwarder using the
  harleypig/mxroute provider's resources directly.
---

# Quick Start

Stand up a working MXroute domain — with a mailbox and a forwarder — using the
`harleypig/mxroute` provider. It mirrors MXroute's own [Quick Setup][mx-quick]
(add a domain → create accounts → connect a client), expressed as Terraform
resources.

**Scope — read this first.** This provider manages the **MXroute account
side** (the domain, mailboxes, forwarders, …) through the MXroute API. It does
**not** manage **DNS records** (MX / SPF / DKIM / DMARC): those live with
whoever hosts your domain's DNS, and you set them there separately (see [DNS
records](#dns-records-set-these-at-your-dns-provider) below). So this guide
gets mail *provisioned* on MXroute; DNS is what makes it *deliver*.

## Prerequisites

- **Terraform >= 1.11.** Mailbox passwords use write-only arguments, which
  Terraform added in 1.11.
- **An MXroute account and API key.** The provider authenticates with your
  server hostname, DirectAdmin username, and API key. Export them so the
  provider (and this guide's commands) can read them:

  ```sh
  export MXROUTE_SERVER="your-server.mxrouting.net"
  export MXROUTE_USERNAME="your-directadmin-user"
  export MXROUTE_API_KEY="…"   # keep this out of shell history / files
  ```

## 1. Configure the provider

The provider is published to the [Terraform Registry][reg], so `terraform
init` installs it automatically from the version constraint in your
configuration.

```hcl
terraform {
  required_version = ">= 1.11"

  required_providers {
    mxroute = {
      source  = "harleypig/mxroute"
      version = "~> 0.3"
    }
  }
}

# Credentials come from the MXROUTE_SERVER / MXROUTE_USERNAME / MXROUTE_API_KEY
# environment variables, so the block is empty.
provider "mxroute" {}
```

## 2. Add a domain

The `mxroute_domain` resource manages a mail domain on your account.

```hcl
resource "mxroute_domain" "primary" {
  domain       = "example.com"
  mail_hosting = true
}
```

> Adding a domain on MXroute requires a one-time **DNS TXT verification** (the
> panel shows the record). That, like all DNS, is done at your DNS provider —
> not through this provider.

## 3. Create an email account

The `mxroute_email_account` resource manages a mailbox. The **password is
write-only** — Terraform never stores it in state — so supply it from a
sensitive variable, never a literal in the config. Bump `password_wo_version`
whenever you change the password so the new value is sent on the next apply.

```hcl
variable "alice_password" {
  description = "Password for the alice mailbox."
  type        = string
  sensitive   = true
}

resource "mxroute_email_account" "alice" {
  domain   = mxroute_domain.primary.domain
  username = "alice"

  password_wo         = var.alice_password
  password_wo_version = 1

  quota = 5120 # MB; omit for unlimited
}
```

Provide the password at apply time via an environment variable (so it never
lands in a file):

```sh
export TF_VAR_alice_password="a-strong-password"
```

> MXroute enforces password complexity at create — use a mix of uppercase,
> lowercase, numbers, and special characters, or the create is rejected.

## 4. Add a forwarder

The `mxroute_forwarder` resource manages an alias that forwards to one or more
addresses. A `postmaster` alias is a good first forwarder — [RFC 2142][rfc2142]
expects every mail domain to accept `postmaster@`.

```hcl
resource "mxroute_forwarder" "postmaster" {
  domain       = mxroute_domain.primary.domain
  alias        = "postmaster"
  destinations = ["alice@example.com"]
}
```

## 5. Initialize and apply

```sh
terraform init     # installs the provider from the Registry
terraform plan     # review what will be created
terraform apply
```

`plan`/`apply` reach the live MXroute API, so the `MXROUTE_*` variables above
must be set. `apply` changes your real account — review the plan first.

## DNS records (set these at your DNS provider)

Provisioning the account is only half the job — mail won't flow until your
domain's DNS points at MXroute. Set these where your DNS is hosted (they are
**not** managed by this provider):

- **MX** — two records, priority 10 and 20, at `your-server.mxrouting.net` /
  `your-server-relay.mxrouting.net` (exact values in your MXroute panel).
- **SPF** — a TXT record: `v=spf1 include:mxroute.com -all`.
- **DKIM** — the domain's key from the panel, as a TXT record at
  `x._domainkey`.
- **DMARC** — a TXT record at `_dmarc` (start at `p=none` and tighten).

The `mxroute_dns` and `mxroute_verification_key` data sources expose the
record values MXroute expects, so you can read them into whatever manages your
DNS. See MXroute's [Quick Setup][mx-quick] for the authoritative values and
propagation notes (DNS changes can take up to 24–48h).

Once DNS resolves, connect a mail client with the standard MXroute ports —
IMAP `993`/`143`, POP3 `995`/`110`, SMTP `465`/`587`/`25` — using the mailbox
you created.

## Next steps

This guide covers the core three resources. The provider also manages
[catch-all policy](email-management.md#catch-all), [spam settings and black- &
white-lists](email-management.md#spam-filtering), domain pointers
(`mxroute_pointer`), and reseller packages/users — see the
[Email Management](email-management.md) guide and each resource's own page.

[mx-quick]: https://docs.mxroute.com/docs/quick-setup.html
[reg]: https://registry.terraform.io/providers/harleypig/mxroute/latest
[rfc2142]: https://www.rfc-editor.org/rfc/rfc2142
