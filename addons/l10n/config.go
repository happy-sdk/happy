// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"golang.org/x/text/language"
)

// l10nLayout selects how an application's own translation keys are laid
// out on disk under its l10n/ directory. It does not affect
// dependencies.json, whose path and nested shape are fixed regardless of
// layout - only the app's own key files move between "one file per
// language" and "one shared file for every language".
type l10nLayout string

const (
	// layoutPerLang is today's (and the default) behavior: one JSON file
	// per language, e.g. l10n/et.json, l10n/de.json.
	layoutPerLang l10nLayout = "per-lang"
	// layoutUnified stores every language's keys in a single shared file,
	// nested one level deeper under each language's own tag - useful for
	// projects that would rather diff/review one file than one per
	// language.
	layoutUnified l10nLayout = "unified"
)

const (
	// defaultPerLangFilenamePattern reproduces the exact filename this
	// package has always used for per-language files (see updateTranslation
	// prior to its config.go split) - changing it would silently move
	// where every existing project's translations are read from and
	// written to.
	defaultPerLangFilenamePattern = "{lang}.json"
	// defaultUnifiedFilename is the shared file used when layout: unified
	// and no explicit filename pattern is configured.
	defaultUnifiedFilename = "translations.json"
)

// l10nFileConfig is addons/l10n's own analog of addons/devel/project's
// Config, scoped to just the "l10n:" top-level key of the shared
// .happy.yaml file. It is decoded directly via goccy/go-yaml, never through
// pkg/settings' Blueprint/Schema/Profile machinery - see
// loadL10nFileConfig's doc comment for why.
type l10nFileConfig struct {
	// Layout is the raw string form of l10nLayout as written in
	// .happy.yaml ("per-lang" or "unified"); empty means "unset, use the
	// default". Kept as a string rather than l10nLayout itself so an
	// unrecognized value decodes without error - isUnified simply treats
	// anything other than the literal string "unified" as per-lang.
	Layout string `yaml:"layout"`
	// Filename is an optional filename pattern overriding the layout's own
	// default (see filenamePattern). For layoutPerLang it must contain a
	// "{lang}" placeholder; for layoutUnified it names the single shared
	// file verbatim.
	Filename string `yaml:"filename"`
}

// happyYAML is the narrow shape addons/l10n reads out of the project's
// shared .happy.yaml file: just its own "l10n:" top-level key. Every
// sibling key (releaser:, linter:, deps:, git:, ignore:, ...) is silently
// ignored by ordinary struct decoding - goccy/go-yaml, unlike
// pkg/settings' Profile.load, never errors on an unrecognized key, so
// there's no risk of this addon's config loader tripping over
// addons/devel/project's (or any other tool's) keys in the same file.
type happyYAML struct {
	L10n l10nFileConfig `yaml:"l10n"`
}

// defaultL10nFileConfig is the configuration used when .happy.yaml is
// missing entirely, has no "l10n:" key, or fails to parse - today's
// existing per-lang behavior, unchanged.
func defaultL10nFileConfig() l10nFileConfig {
	return l10nFileConfig{Layout: string(layoutPerLang)}
}

// loadL10nFileConfig reads the "l10n:" key out of moduleRoot's .happy.yaml,
// if any. It never returns an error: a missing file, a missing "l10n:" key,
// or a malformed YAML document all fall back to defaultL10nFileConfig
// exactly as if nothing had been configured. This is deliberate - unlike
// addons/devel/project's Config, which is central to that tool's own
// operation and so must fail loudly on a broken project config,
// addons/l10n's file layout config is a narrow, optional refinement of
// where translations already live; a typo or a half-written .happy.yaml
// should never prevent `l10n report`/`translate`/`tui` from working
// against the plain default layout.
//
// pkg/settings' Blueprint/Schema/Profile machinery (as used by
// addons/devel/project) is not used here: Profile.load hard-errors on any
// decoded YAML key that isn't declared in that schema's own settings map,
// and a real project's .happy.yaml commonly carries releaser:/linter:/
// deps:/git:/ignore: keys owned by addons/devel/project's schema. There is
// no way to ask that machinery to decode just one subset of keys and
// ignore the rest, so this package decodes the file itself instead.
func loadL10nFileConfig(moduleRoot string) (l10nFileConfig, error) {
	cnf := defaultL10nFileConfig()

	data, err := os.ReadFile(filepath.Join(moduleRoot, ".happy.yaml"))
	if err != nil {
		return cnf, nil
	}

	var doc happyYAML
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return cnf, nil
	}

	if doc.L10n.Layout != "" {
		cnf.Layout = doc.L10n.Layout
	}
	cnf.Filename = doc.L10n.Filename

	return cnf, nil
}

// isUnified reports whether c selects the unified (one shared file for
// every language) layout rather than the default per-lang one.
func (c l10nFileConfig) isUnified() bool {
	return l10nLayout(c.Layout) == layoutUnified
}

// filenamePattern resolves c.Filename to the pattern that should actually
// be used, applying the right default for the selected layout when
// nothing explicit was configured.
func (c l10nFileConfig) filenamePattern() string {
	if c.Filename != "" {
		return c.Filename
	}
	if c.isUnified() {
		return defaultUnifiedFilename
	}
	return defaultPerLangFilenamePattern
}

// appFilePath resolves the on-disk path of the app's own translation file
// for lang, under l10nDir (the project's l10n/ directory). For the unified
// layout every language resolves to the same shared path; the lang
// parameter still has to be accepted so callers don't need to branch on
// layout themselves.
func (c l10nFileConfig) appFilePath(l10nDir string, lang language.Tag) string {
	pattern := c.filenamePattern()
	if c.isUnified() {
		return filepath.Join(l10nDir, pattern)
	}
	return filepath.Join(l10nDir, strings.ReplaceAll(pattern, "{lang}", lang.String()))
}
