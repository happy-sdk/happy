// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

// Package mcp serves a workspace over the Model Context Protocol.
//
// It discovers the repositories checked out in a workspace, reads the agent
// manifest each one ships, and exposes their tools and skills namespaced by
// repository. A client is configured once; what it can do then follows the
// workspace, so adding a tool to a repository never means reconfiguring
// anything.
package mcp

import (
	"errors"

	"github.com/happy-sdk/happy/sdk/addon"
)

var Error = errors.New("mcp")

// Addon provides the mcp command tree, the same way the l10n addon provides
// its own.
func Addon() *addon.Addon {
	return addon.New("MCP").
		WithConfig(addon.Config{
			Slug: "mcp",
		}).
		ProvideCommands(Command())
}
