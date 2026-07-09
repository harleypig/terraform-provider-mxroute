# Security Policy

## Supported versions

This provider follows semantic versioning (current major: 1). Security fixes
are made against the latest release only; please upgrade to the newest tag
before reporting.

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub
issues.**

Report privately through GitHub's
[security advisories](https://github.com/harleypig/terraform-provider-mxroute/security/advisories/new)
("Report a vulnerability"). Include a description, reproduction steps, and the
affected version. You'll get an acknowledgement, and a fix and disclosure will
be coordinated with you.

## Handling of credentials

- The provider's MXroute API key (`api_key` / `MXROUTE_API_KEY`) is a
  `Sensitive` value and is never written to logs.
- Mailbox and reseller passwords are **write-only** attributes: they are sent
  to the API but never stored in Terraform state.
- Never commit real credentials. Local acceptance tests read them from the
  environment (`MXROUTE_SERVER` / `MXROUTE_USERNAME` / `MXROUTE_API_KEY`), and
  the repository runs secret scanning (`gitleaks`, `detect-private-key`) in
  pre-commit and CI.
