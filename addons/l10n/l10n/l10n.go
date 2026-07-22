// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package l10n loads the l10n addon's own bundle (its command/flag help
// text, in locales/*.json) - nothing else. Every package in this monorepo
// that provides translations keeps its loader in a dedicated "l10n"
// subpackage like this one, so a maintainer looking for where a package's
// translations get embedded and registered always knows exactly where to
// look, rather than having to search through a larger, multi-purpose
// package for it.
package l10n

import (
	"embed"

	"github.com/happy-sdk/happy/pkg/i18n"
)

//go:embed locales/*
var locales embed.FS

// Embed, not MustEmbed: addons/ (unlike happy itself, sdk/, and pkg/) may
// carry third-party dependencies, so a broken bundle here should let the
// embedding application decide whether to continue or exit cleanly with
// its own error, rather than this addon unilaterally panicking a host
// application it doesn't control.
func init() { i18n.Embed(locales) }
