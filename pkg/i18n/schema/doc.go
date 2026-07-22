// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package schema defines the versioned, on-disk JSON shape of happy-sdk
// translation files - the *.json files github.com/happy-sdk/happy/pkg/i18n
// loads via a translations *FS or RegisterTranslations - and dispatches a
// parsed file to whichever version it declares.
//
// pkg/i18n may depend only on the Go standard library and golang.org/x/text
// (the Go team's own extended-stdlib text/i18n packages) - never a
// third-party module - so this package (and its v1/v2/... subpackages)
// implement their own minimal validation rather than pulling in a
// general-purpose JSON Schema validator. Each subpackage's own schema.json
// is for humans and editors only - point a translation file's "$schema" at
// one for autocomplete/validation in any JSON-Schema-aware editor - never
// consumed at runtime.
//
// # Versions
//
// Each schema version lives in its own subpackage (v1, v2, ...), and
// defines its own document shape independently; this package's only job is
// Parse: read the reserved KeyVersion key (if any) and hand the rest of
// the document to the matching subpackage, normalizing whatever it
// returns into the version-agnostic Document every version produces.
//
//   - v1 is github.com/happy-sdk/happy/pkg/i18n's original, unversioned
//     format: no KeyVersion key at all, one locale per file, filename
//     supplies that locale.
//   - v2 is self-describing: a required "bundle" identifier and "locales"
//     object (one entry per locale it carries - one or many) replace both
//     the implicit root-key convention and the filename-as-locale
//     convention.
//
// A document declaring a version this package doesn't recognize (above
// the newest subpackage that exists, or simply not an integer) is reported
// as an error rather than guessed at - see VersionError.
package schema
