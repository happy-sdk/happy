// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"golang.org/x/text/language"
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

func (f *FS) readRoot() ([]fs.DirEntry, error) {
	return f.content.ReadDir(f.prefix)
}

func (f *FS) load(lang language.Tag, dir string) error {
	translationFiles, err := f.content.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("i18n(%s) loading translations from fs failed: %s", lang.String(), err.Error())
	}

	for _, file := range translationFiles {
		if file.IsDir() {
			return fmt.Errorf("i18n(%s): expected translation file in lang dir got directory: %s", lang.String(), file.Name())
		}
		content, err := f.content.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return fmt.Errorf("i18n(%s): reading translation file %s failed: %s", lang.String(), file.Name(), err.Error())
		}

		var translations map[string]any
		if err := json.Unmarshal(content, &translations); err != nil {
			return fmt.Errorf("i18n(%s): could not parse translation file %s: %s", lang.String(), file.Name(), err.Error())
		}
		if err := RegisterTranslations(lang, translations); err != nil {
			return err
		}
	}
	return nil
}
