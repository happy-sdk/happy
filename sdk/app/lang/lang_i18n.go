// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build i18n

package lang

import (
	"embed"

	"github.com/happy-sdk/happy/pkg/i18n"
)

//go:embed *
var translations embed.FS

func init() {
	_ = i18n.RegisterTranslationsFS(
		i18n.NewFS(translations).WithPrefix("."),
	)
}
