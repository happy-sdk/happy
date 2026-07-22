// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package schema

import (
	"errors"
	"fmt"

	v1 "github.com/happy-sdk/happy/pkg/i18n/schema/v1"
	v2 "github.com/happy-sdk/happy/pkg/i18n/schema/v2"
	"golang.org/x/text/language"
)

const (
	// KeyVersion is the reserved top-level key declaring which schema
	// subpackage a document was written against, e.g. `"version": 2`. A
	// document with no such key at all is schema version 1 - see package
	// doc comment.
	KeyVersion = "version"

	// V2 is the newest schema version this package knows how to dispatch.
	// There's no corresponding V1 constant: V1 isn't a value KeyVersion
	// ever actually holds - its absence is what means "version 1".
	V2 = 2

	// CurrentVersion is the schema version this package (and pkg/i18n's
	// own bundled translations) target.
	CurrentVersion = V2
)

// Document is the version-agnostic result of parsing a translation file,
// however many locales or whichever schema version it declared. Bundle is
// "" for a v1 document - v1 has no single explicit bundle identifier of
// its own; its top-level keys are root/bundle keys directly, a
// registration-time concern package i18n's own registerTranslations still
// owns. Source is the zero language.Tag and Notes is nil for a v1 document
// too - v1 predates both a bundle's declared source locale and per-locale
// translator notes entirely (see v2.KeySource/v2.KeyLocaleNotes).
type Document struct {
	Bundle  string
	Source  language.Tag
	Locales map[language.Tag]map[string]any
	Notes   map[language.Tag]string
}

// ErrUnsupportedVersion is returned by Parse when a translation file
// declares a KeyVersion this build of the package doesn't recognize -
// above CurrentVersion, or not a recognizable integer at all. Test with
// errors.Is; VersionError carries the specifics.
var ErrUnsupportedVersion = errors.New("unsupported translation file schema version")

// VersionError reports why a file's declared schema version was rejected.
// Got is the parsed version, or -1 if the KeyVersion value wasn't a
// recognizable integer at all.
type VersionError struct {
	Got int
}

func (e *VersionError) Error() string {
	return fmt.Sprintf("%s: %d (newest supported is %d)", ErrUnsupportedVersion, e.Got, CurrentVersion)
}

func (e *VersionError) Unwrap() error { return ErrUnsupportedVersion }

// Parse dispatches raw - a translation file's top-level parsed JSON object
// - to the schema version it declares (see KeyVersion), normalizing the
// result into a Document. lang is used only if raw turns out to be an
// unversioned (schema version 1) document, which carries no locale of its
// own - typically the loading FS's own filename-derived language.Tag, or
// language.Und if there is none.
func Parse(lang language.Tag, raw map[string]any) (*Document, error) {
	versionVal, hasVersion := raw[KeyVersion]
	if !hasVersion {
		doc, err := v1.Parse(lang, raw)
		if err != nil {
			return nil, err
		}
		return &Document{Locales: map[language.Tag]map[string]any{doc.Lang: doc.Body}}, nil
	}

	n, ok := toInt(versionVal)
	if !ok {
		return nil, &VersionError{Got: -1}
	}

	switch n {
	case V2:
		body := make(map[string]any, len(raw))
		for k, v := range raw {
			if k == KeyVersion {
				continue
			}
			body[k] = v
		}
		doc, err := v2.Parse(body)
		if err != nil {
			return nil, err
		}
		return &Document{Bundle: doc.Bundle, Source: doc.Source, Locales: doc.Locales, Notes: doc.Notes}, nil
	default:
		return nil, &VersionError{Got: n}
	}
}

// toInt reports v's integer value, if v is one of the numeric types a JSON
// number can decode into for an `any`-typed field.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}
