// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

//go:build i18n

package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"golang.org/x/text/language"
)

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
