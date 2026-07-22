// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package schema

import (
	"errors"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestParse_UnversionedIsV1SingleLocale(t *testing.T) {
	doc, err := Parse(language.English, map[string]any{
		"app.description": "hello",
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "", doc.Bundle)
	testutils.Equal(t, 1, len(doc.Locales))
	testutils.Equal(t, "hello", doc.Locales[language.English]["app.description"])
}

func TestParse_UnversionedDefaultsToPassedLang(t *testing.T) {
	doc, err := Parse(language.Und, map[string]any{"k": "v"})
	testutils.NoError(t, err)
	if _, ok := doc.Locales[language.Und]; !ok {
		t.Fatal("expected the document's single locale to be keyed under the passed (Und) language")
	}
}

func TestParse_V2Dispatch(t *testing.T) {
	doc, err := Parse(language.Und, map[string]any{
		"version": float64(2),
		"bundle":  "app",
		"locales": map[string]any{
			"en": map[string]any{
				"notes": "scope note",
				"keys":  map[string]any{"greeting": "hello"},
			},
			"de": map[string]any{"keys": map[string]any{"greeting": "hallo"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "app", doc.Bundle)
	testutils.Equal(t, "scope note", doc.Notes[language.English])
	testutils.Equal(t, 2, len(doc.Locales))
	testutils.Equal(t, "hello", doc.Locales[language.English]["greeting"])
	testutils.Equal(t, "hallo", doc.Locales[language.German]["greeting"])
}

func TestParse_VersionKeyStrippedFromV2Body(t *testing.T) {
	doc, err := Parse(language.Und, map[string]any{
		"version": float64(2),
		"bundle":  "app",
		"locales": map[string]any{"en": map[string]any{"keys": map[string]any{"a": "b"}}},
	})
	testutils.NoError(t, err)
	if _, ok := doc.Locales[language.English]["version"]; ok {
		t.Fatal("the reserved version key must never leak into a locale's translation tree")
	}
}

func TestParse_VersionTooNewIsRejected(t *testing.T) {
	_, err := Parse(language.Und, map[string]any{"version": float64(CurrentVersion + 1)})
	if err == nil {
		t.Fatal("expected an error for a version newer than CurrentVersion")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected errors.Is(err, ErrUnsupportedVersion), got %v", err)
	}
}

func TestParse_VersionNotAnIntegerIsRejected(t *testing.T) {
	_, err := Parse(language.Und, map[string]any{"version": "2"})
	if err == nil {
		t.Fatal("expected an error for a non-integer version")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected errors.Is(err, ErrUnsupportedVersion), got %v", err)
	}
}

func TestParse_V2EnvelopeErrorPropagates(t *testing.T) {
	_, err := Parse(language.Und, map[string]any{
		"version": float64(2),
		// missing required "bundle"
		"locales": map[string]any{"en": map[string]any{"keys": map[string]any{"a": "b"}}},
	})
	if err == nil {
		t.Fatal("expected the v2 envelope validation error to propagate")
	}
}
