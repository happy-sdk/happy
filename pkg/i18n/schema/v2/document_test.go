// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package v2

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestParse_Minimal(t *testing.T) {
	doc, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "app", doc.Bundle)
	testutils.Equal(t, "hello", doc.Locales[language.English]["greeting"])
}

func TestParse_DottedBundle(t *testing.T) {
	doc, err := Parse(map[string]any{
		"bundle": "com.github.happy-sdk.happy.pkg.i18n",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"name": "English"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "com.github.happy-sdk.happy.pkg.i18n", doc.Bundle)
}

func TestParse_MultipleLocalesInOneDocument(t *testing.T) {
	doc, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
			"de": map[string]any{"keys": map[string]any{"greeting": "hallo"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, 2, len(doc.Locales))
	testutils.Equal(t, "hallo", doc.Locales[language.German]["greeting"])
}

func TestParse_PerLocaleNotes(t *testing.T) {
	doc, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{
				"notes": "context for English translators",
				"keys":  map[string]any{"greeting": "hello"},
			},
			"de": map[string]any{"keys": map[string]any{"greeting": "hallo"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "context for English translators", doc.Notes[language.English])
	if _, ok := doc.Notes[language.German]; ok {
		t.Fatal("expected no notes entry for a locale that declared none")
	}
}

func TestParse_Source(t *testing.T) {
	doc, err := Parse(map[string]any{
		"bundle": "app",
		"source": "de",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, language.German, doc.Source)
}

func TestParse_SourceDefaultsToEnglish(t *testing.T) {
	doc, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, DefaultSource, doc.Source)
	testutils.Equal(t, language.English, doc.Source)
}

func TestParse_InvalidSourceIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle": "app",
		"source": "not-a-locale-!!",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an invalid source locale")
	}
}

func TestParse_MissingBundleIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{
		"locales": map[string]any{"en": map[string]any{"keys": map[string]any{"a": "b"}}},
	})
	if err == nil {
		t.Fatal("expected an error for a missing bundle")
	}
}

func TestParse_InvalidBundleIsRejected(t *testing.T) {
	for _, bad := range []string{"", "Com.Example", "1com.example", ".com.example", "com..example", "com.example."} {
		_, err := Parse(map[string]any{
			"bundle":  bad,
			"locales": map[string]any{"en": map[string]any{"keys": map[string]any{"a": "b"}}},
		})
		if err == nil {
			t.Fatalf("expected an error for invalid bundle %q", bad)
		}
	}
}

func TestParse_MissingLocalesIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{"bundle": "app"})
	if err == nil {
		t.Fatal("expected an error for missing locales")
	}
}

func TestParse_EmptyLocalesIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{"bundle": "app", "locales": map[string]any{}})
	if err == nil {
		t.Fatal("expected an error for empty locales")
	}
}

func TestParse_InvalidLocaleCodeIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle":  "app",
		"locales": map[string]any{"not-a-locale-!!": map[string]any{"keys": map[string]any{"a": "b"}}},
	})
	if err == nil {
		t.Fatal("expected an error for an unparseable locale code")
	}
}

func TestParse_LocaleMissingKeysIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle":  "app",
		"locales": map[string]any{"en": map[string]any{"notes": "only notes, no keys"}},
	})
	if err == nil {
		t.Fatal("expected an error for a locale entry missing its required \"keys\" object")
	}
}

func TestParse_LocaleUnknownKeyIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{
				"keys":    map[string]any{"a": "b"},
				"unknown": "x",
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown key within a locale entry")
	}
}

func TestParse_UnknownTopLevelKeyIsRejected(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle":    "app",
		"locales":   map[string]any{"en": map[string]any{"keys": map[string]any{"a": "b"}}},
		"unknown":   "x",
		"extraKey2": "y",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown top-level key")
	}
}

func TestParse_InvalidKeyNameIsRejected(t *testing.T) {
	for _, bad := range []string{"CamelCase", "with-dash", "with space", "with.dot"} {
		_, err := Parse(map[string]any{
			"bundle":  "app",
			"locales": map[string]any{"en": map[string]any{"keys": map[string]any{bad: "value"}}},
		})
		if err == nil {
			t.Fatalf("expected an error for invalid key name %q", bad)
		}
	}
}

func TestParse_MessageKeyIsExemptFromKeyNaming(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{
				"keys": map[string]any{
					"greeting": map[string]any{
						"$message": map[string]any{"msg": "hello {name:s}"},
					},
				},
			},
		},
	})
	testutils.NoError(t, err)
}

func TestParse_NestedGroupsAreValidated(t *testing.T) {
	_, err := Parse(map[string]any{
		"bundle": "app",
		"locales": map[string]any{
			"en": map[string]any{
				"keys": map[string]any{
					"errors": map[string]any{
						"BadKey": "value",
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an invalid nested key name")
	}
}
