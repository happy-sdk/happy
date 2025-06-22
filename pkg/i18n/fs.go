// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"embed"
)

type FS struct {
	// Prefix directory path where language subdirectories are stored.
	// Default is "lang".
	prefix  string
	content embed.FS
}

func NewFS(fs embed.FS) *FS {
	return &FS{
		prefix:  "lang",
		content: fs,
	}
}

// WithPrefix sets the prefix directory path where
// language subdirectories are stored.
func (f *FS) WithPrefix(prefix string) *FS {
	f.prefix = prefix
	return f
}
