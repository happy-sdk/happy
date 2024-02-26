// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import "github.com/happy-sdk/happy/pkg/cli/ansicolor"

type Builder struct {
	brand *Brand
}

func (b *Builder) Build() (*Brand, error) {
	return b.brand, nil
}

func (b *Builder) WithANSI(ansi ansicolor.Theme) *Builder {
	b.brand.ansi = ansi
	return b
}
