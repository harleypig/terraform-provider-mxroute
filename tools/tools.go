//go:build generate

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

// Format the example Terraform used in the docs (needs a local terraform binary).
//go:generate terraform fmt -recursive ../examples/

// Generate the Registry documentation from the provider schema + examples.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name mxroute
