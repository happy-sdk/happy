// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import "github.com/happy-sdk/happy/pkg/cli/ansicolor"

type Brand struct {
	info Info
	ansi ansicolor.Theme
}

type Info struct {
	Name        string
	Version     string
	Slug        string
	Description string
}

func (b *Brand) Info() Info {
	return b.info
}

func (b *Brand) ANSI() ansicolor.Theme {
	return b.ansi
}
