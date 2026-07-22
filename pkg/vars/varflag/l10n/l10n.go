// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package l10n

import (
	"embed"

	"github.com/happy-sdk/happy/pkg/i18n"
)

//go:embed locales/*
var locales embed.FS

// MustEmbed, not Embed: everything under pkg/ has no third-party
// dependencies and only ever releases modules whose bundles are fully
// valid - a broken bundle here must fail loudly at process start, not
// silently warn and leave every string in this package rendering only its
// fallback.
func init() { i18n.MustEmbed(locales) }
