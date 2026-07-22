// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package v2

import (
	"fmt"
	"regexp"

	v1 "github.com/happy-sdk/happy/pkg/i18n/schema/v1"
	"golang.org/x/text/language"
)

// Reserved top-level keys a schema version 2 document may hold (besides
// the parent schema package's own KeyVersion, already removed from body by
// the time Parse sees it).
const (
	// KeyBundle is the required root/bundle identifier - see bundleRe.
	KeyBundle = "bundle"
	// KeySource is the bundle's source locale - the language its
	// translations are authored/maintained in, and every other locale is
	// translated from. Optional; defaults to DefaultSource ("en") when
	// absent. A bundle split across several per-locale files (see package
	// doc comment) should repeat the same KeySource in each one.
	KeySource = "source"
	// KeyLocales holds one entry per locale this document carries - see
	// Parse and KeyLocaleNotes/KeyLocaleKeys.
	KeyLocales = "locales"
)

// Reserved keys within a single KeyLocales entry.
const (
	// KeyLocaleNotes is optional, free-text, translator/tooling-facing
	// context for this locale's own translations - e.g. anything specific
	// to how this particular language should render them. Not the same
	// thing as a Message's own "description" (see v1.Description), which
	// documents one message, not a whole locale.
	KeyLocaleNotes = "notes"
	// KeyLocaleKeys holds the locale's actual translation tree.
	KeyLocaleKeys = "keys"
)

// DefaultSource is the source locale a document is assumed to use when it
// doesn't declare KeySource explicitly.
var DefaultSource = language.English

// bundleRe accepts either a single bare identifier (e.g. "app") or a
// dot-separated, reverse-DNS-style path (e.g.
// "com.github.happy-sdk.happy.pkg.i18n") - hyphens allowed within a
// segment, matching real Go module path components like "happy-sdk".
var bundleRe = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)*$`)

// keyNameRe is the only shape a translation key (a path segment leading to
// a message, not v1.MessageKey itself) may take.
var keyNameRe = regexp.MustCompile(`^[a-z0-9_]+$`)

// Document is a fully parsed and structurally validated schema version 2
// translation file. Notes is per-locale (a locale's own translator/tooling
// context, e.g. anything specific to how that language should render these
// entries), not per-bundle - a language absent from Notes simply declared
// no KeyLocaleNotes of its own.
type Document struct {
	Bundle  string
	Source  language.Tag
	Locales map[language.Tag]map[string]any
	Notes   map[language.Tag]string
}

// Parse parses and structurally validates body - a translation file's
// top-level JSON object with its reserved "version" key already removed -
// per schema version 2 (see package doc comment for the full shape).
func Parse(body map[string]any) (*Document, error) {
	for k := range body {
		switch k {
		case KeyBundle, KeySource, KeyLocales:
		default:
			return nil, fmt.Errorf("unknown top-level key %q (want %q, %q, or %q)", k, KeyBundle, KeySource, KeyLocales)
		}
	}

	bundle, ok := body[KeyBundle].(string)
	if !ok || bundle == "" {
		return nil, fmt.Errorf("missing or empty required %q field", KeyBundle)
	}
	if !bundleRe.MatchString(bundle) {
		return nil, fmt.Errorf("%q is not a valid %q (want a bare word or dot-separated reverse-DNS-style path, e.g. %q)",
			bundle, KeyBundle, "com.github.example.app")
	}

	source := DefaultSource
	if sourceRaw, ok := body[KeySource]; ok {
		code, ok := sourceRaw.(string)
		if !ok {
			return nil, fmt.Errorf("%q must be a string locale code", KeySource)
		}
		lang, err := language.Parse(code)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid %q: %w", code, KeySource, err)
		}
		source = lang
	}

	localesRaw, ok := body[KeyLocales].(map[string]any)
	if !ok || len(localesRaw) == 0 {
		return nil, fmt.Errorf("missing or empty required %q object", KeyLocales)
	}

	locales := make(map[language.Tag]map[string]any, len(localesRaw))
	notes := make(map[language.Tag]string)
	for code, entryRaw := range localesRaw {
		lang, err := language.Parse(code)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid locale: %w", code, err)
		}

		entry, ok := entryRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("locale %q must be an object", code)
		}
		for k := range entry {
			switch k {
			case KeyLocaleNotes, KeyLocaleKeys:
			default:
				return nil, fmt.Errorf("locale %q: unknown key %q (want %q or %q)", code, k, KeyLocaleNotes, KeyLocaleKeys)
			}
		}

		tree, ok := entry[KeyLocaleKeys].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("locale %q is missing its required %q object", code, KeyLocaleKeys)
		}
		if err := validateKeys(tree); err != nil {
			return nil, err
		}
		locales[lang] = tree

		if noteText, ok := entry[KeyLocaleNotes].(string); ok && noteText != "" {
			notes[lang] = noteText
		}
	}

	return &Document{Bundle: bundle, Source: source, Locales: locales, Notes: notes}, nil
}

// validateKeys recursively rejects any translation key that isn't
// lowercase a-z, 0-9, or "_". v1.MessageKey ("$message") is the one
// exception - and its value isn't recursed into at all under this rule:
// everything inside it is v1.ParseMessage's own fixed schema (msg,
// fragments, description, kind, arg, forms, zero, one, ...), never a
// developer-chosen key name.
func validateKeys(tree map[string]any) error {
	for key, value := range tree {
		if key == v1.MessageKey {
			continue
		}
		if !keyNameRe.MatchString(key) {
			return fmt.Errorf(`%q is not a valid translation key (want only lowercase a-z, 0-9, or "_")`, key)
		}
		if nested, ok := value.(map[string]any); ok {
			if err := validateKeys(nested); err != nil {
				return err
			}
		}
	}
	return nil
}
