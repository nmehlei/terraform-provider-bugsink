//go:build tools

// Package tools tracks development-only tool dependencies in go.mod
// without pulling them into the provider's build. See
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
