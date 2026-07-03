# Examples

This directory holds examples used mostly for documentation, but they can also
be run/tested manually via the Terraform CLI.

The `tfplugindocs` generation tool looks for files in the locations below by
default; all other `*.tf` files are ignored by the docs tool (useful for
runnable/testable examples whose parts aren't all relevant to the docs):

- **provider/provider.tf** — the provider index page
- **data-sources/`full data source name`/data-source.tf** — the named
  data-source page
- **resources/`full resource name`/resource.tf** — the named resource page
