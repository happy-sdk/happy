// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package v1

import "golang.org/x/text/language"

// Document is a schema version 1 translation file: a single locale's
// translation tree, exactly as found in the file - Body's own top-level
// keys are root/bundle keys (or a flat, unprefixed tree; see package
// i18n's registerTranslations, which still owns that distinction - it's a
// registration-time concern, not part of this schema).
type Document struct {
	Lang language.Tag
	Body map[string]any
}

// Parse builds a Document from raw, a translation file's top-level parsed
// JSON object. Since a schema version 1 file carries no locale of its own,
// lang must come from outside - typically the loading FS's own filename-
// derived language.Tag - or language.Und if the caller has none.
func Parse(lang language.Tag, raw map[string]any) (*Document, error) {
	return &Document{Lang: lang, Body: raw}, nil
}
