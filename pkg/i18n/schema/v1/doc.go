// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package v1 defines schema version 1: github.com/happy-sdk/happy/pkg/i18n's
// original translation file shape, from before schema versioning existed.
// A translation file in this shape declares no reserved top-level key at
// all (see the parent schema package's KeyVersion) - its own top-level
// keys are root/bundle keys directly, exactly as registerTranslations has
// always treated them, and its locale comes entirely from outside (the
// loading FS's own filename, or whatever a direct RegisterTranslations
// caller passes) rather than from anything in the file itself.
//
// This package also defines the "$message" value shape (Message,
// Fragment, Description - plural/ordinal/select CLDR-aware rich
// translation entries with translator-facing documentation), which is NOT
// itself versioned independently: schema version 2 (see the sibling v2
// package) reuses this exact same shape unchanged for its own translation
// values, via ParseMessage. Only the surrounding document envelope differs
// between schema versions.
package v1
