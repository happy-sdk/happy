// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

//go:build i18n

package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

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
		if err := f.loadFile(lang, filepath.Join(dir, file.Name())); err != nil {
			return err
		}

	}
	return nil
}

func (f *FS) loadFile(lang language.Tag, fpath string) error {
	content, err := f.content.ReadFile(fpath)
	name := filepath.Base(fpath)
	if err != nil {
		return fmt.Errorf("i18n(%s): reading translation file %s failed: %s", lang.String(), name, err.Error())
	}

	var translations map[string]any
	if err := json.Unmarshal(content, &translations); err != nil {
		return fmt.Errorf("i18n(%s): could not parse translation file %s: %s", lang.String(), name, err.Error())
	}
	if err := RegisterTranslations(lang, translations); err != nil {
		return err
	}
	return nil
}

func registerTranslationsFS(fs *FS) (res error) {
	defer func() {
		if res != nil {
			slog.Warn(res.Error())
		}
	}()
	langDirs, err := fs.readRoot()
	if err != nil {
		res = fmt.Errorf("i18n loading translations from fs failed: %s", err.Error())
		return
	}
	for _, entry := range langDirs {
		name := entry.Name()
		if !entry.IsDir() {
			if filepath.Ext(name) != ".json" {
				continue
			}
			langStr := strings.TrimSuffix(filepath.Base(name), ".json")
			lang, err := language.Parse(langStr)
			if err != nil {
				res = fmt.Errorf("i18n parsing language tag from file %s failed: %s", name, err.Error())
				return
			}

			if err := fs.loadFile(lang, filepath.Join(fs.prefix, name)); err != nil {
				res = err
				return
			}
			continue
		}
		lang, err := language.Parse(name)
		if err != nil {
			res = fmt.Errorf("i18n parsing language tag from dir %s failed: %s", name, err.Error())
			return
		}
		if err := fs.load(lang, filepath.Join(fs.prefix, name)); err != nil {
			res = err
			return
		}
	}

	return nil
}
