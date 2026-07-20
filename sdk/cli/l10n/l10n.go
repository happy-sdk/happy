// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package l10n

import (
	"embed"

	"github.com/happy-sdk/happy/pkg/i18n"
)

//go:embed *.json
var translations embed.FS

func init() {
	_ = i18n.RegisterTranslationsFS(
		i18n.NewFS(translations).WithPrefix("."),
	)
}
