// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"embed"
	"encoding/json/v2"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/i18n/schema"
	"golang.org/x/text/language"
)

// FS adapts an embed.FS holding translation files into the shape
// RegisterTranslationsFS (and, through it, Embed/EmbedIssues/MustEmbed)
// expects: either a flat directory of "<lang>.json" files, or one
// subdirectory per language each holding one or more translation files.
// Construct one with NewFS.
type FS struct {
	// prefix is the directory path within content where language
	// subdirectories (or "<lang>.json" files) are stored. Default is
	// "locales" - the one convention every bundle in this monorepo uses (a
	// package named "l10n" holds Go loader code only; its actual
	// translation content lives in a nested "locales" subdirectory, not a
	// second "l10n" - see Embed/MustEmbed). Change it with WithPrefix.
	prefix  string
	content embed.FS
}

// NewFS wraps fsys as an *FS rooted at fsys's "locales" subdirectory - the
// layout Embed, EmbedIssues, and MustEmbed all assume. Use WithPrefix if
// fsys's translation content lives somewhere else.
func NewFS(fs embed.FS) *FS {
	return &FS{
		prefix:  "locales",
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
		return ErrReadDir.WithArgs("dir", dir, "cause", err.Error())
	}

	for _, file := range translationFiles {
		if file.IsDir() {
			return ErrUnexpectedDirectory.WithArgs("name", file.Name(), "lang", lang.String())
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
		return ErrReadFile.WithArgs("file", name, "cause", err.Error())
	}

	var translations map[string]any
	if err := json.Unmarshal(content, &translations); err != nil {
		return ErrParseFile.WithArgs("file", name, "cause", err.Error())
	}

	if err := RegisterTranslations(lang, translations); err != nil {
		return err
	}
	return nil
}

// registerTranslationsFS is a pure primitive: it only ever returns an
// error, never logs one itself. Logging (or not) is a policy decision that
// belongs to whichever caller actually knows what a failure here should
// mean - Embed warns, MustEmbed panics, Initialize collects it into a
// returned InitIssue - not something this package should decide on their
// behalf by logging out from underneath them.
func registerTranslationsFS(fs *FS) (res error) {
	langDirs, err := fs.readRoot()
	if err != nil {
		res = ErrReadDir.WithArgs("dir", fs.prefix, "cause", err.Error())
		return
	}
	for _, entry := range langDirs {
		name := entry.Name()
		if !entry.IsDir() {
			if filepath.Ext(name) != ".json" {
				continue
			}
			fpath := filepath.Join(fs.prefix, name)
			langStr := strings.TrimSuffix(filepath.Base(name), ".json")
			lang, langErr := language.Parse(langStr)
			if langErr != nil {
				// The file's own name doesn't encode a locale. That's
				// fine for a self-describing schema version 2 document
				// (see schema.KeyVersion) - it needs no filename hint at
				// all - but a schema version 1 document has no locale of
				// its own, so an unparseable name there is a real error,
				// exactly as it always was.
				if !fs.declaresOwnVersion(fpath) {
					res = ErrParseLanguageTag.WithArgs("source", name, "cause", langErr.Error())
					return
				}
				lang = language.Und
			}

			if err := fs.loadFile(lang, fpath); err != nil {
				res = err
				return
			}
			continue
		}
		lang, err := language.Parse(name)
		if err != nil {
			res = ErrParseLanguageTag.WithArgs("source", name, "cause", err.Error())
			return
		}
		if err := fs.load(lang, filepath.Join(fs.prefix, name)); err != nil {
			res = err
			return
		}
	}

	return nil
}

// declaresOwnVersion reports whether the file at fpath is valid JSON
// declaring the reserved schema.KeyVersion key - i.e. whether it's a
// self-describing document that doesn't need a filename-derived locale at
// all. Any error reading or parsing it here is not itself reported - the
// caller falls back to its own filename-parse error instead, which is the
// more informative failure for what's actually the common case (a typo'd
// language code).
func (f *FS) declaresOwnVersion(fpath string) bool {
	content, err := f.content.ReadFile(fpath)
	if err != nil {
		return false
	}
	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		return false
	}
	_, ok := raw[schema.KeyVersion]
	return ok
}
