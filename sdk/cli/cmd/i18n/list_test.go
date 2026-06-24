// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

func TestProcessEntriesForDisplay(t *testing.T) {
	entries := []i18n.TranslationEntry{
		{
			Key:      "app.fully.translated",
			Fallback: "Hello",
			Translations: map[language.Tag]string{
				language.French: "Bonjour",
			},
		},
		{
			Key:      "app.missing.french",
			Fallback: "World",
			Translations: map[language.Tag]string{
				language.German: "Welt",
			},
		},
	}
	langs := []language.Tag{language.French, language.German}

	t.Run("all languages, not missing-only", func(t *testing.T) {
		got := processEntriesForDisplay(entries, langs, language.Und, language.English, false)
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(got))
		}
		// Sorted by key: "app.fully.translated" < "app.missing.french"
		if got[0].Key != "app.fully.translated" || got[0].Status != "missing" {
			// app.fully.translated is missing German, so still "missing" overall
			t.Errorf("entry 0 = %+v", got[0])
		}
	})

	t.Run("missing-only filters fully-translated entries", func(t *testing.T) {
		allTranslated := []i18n.TranslationEntry{
			{
				Key:      "app.complete",
				Fallback: "Hi",
				Translations: map[language.Tag]string{
					language.French: "Salut",
					language.German: "Hallo",
				},
			},
		}
		got := processEntriesForDisplay(allTranslated, langs, language.Und, language.English, true)
		if len(got) != 0 {
			t.Errorf("expected fully-translated entry to be filtered out, got %d entries", len(got))
		}
	})

	t.Run("target language narrows the check to one language", func(t *testing.T) {
		got := processEntriesForDisplay(entries, langs, language.French, language.English, false)
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(got))
		}
		for _, ks := range got {
			if len(ks.MissingLangs)+len(ks.TranslatedLangs) != 1 {
				t.Errorf("expected exactly 1 language checked for key %q, got missing=%v translated=%v",
					ks.Key, ks.MissingLangs, ks.TranslatedLangs)
			}
		}
	})

	t.Run("fallback language checked via Fallback field", func(t *testing.T) {
		got := processEntriesForDisplay(entries, []language.Tag{language.English}, language.Und, language.English, false)
		for _, ks := range got {
			if ks.Status != "ok" {
				t.Errorf("expected fallback language to always be translated (Fallback is non-empty), got %+v", ks)
			}
		}
	})

	t.Run("results sorted by key", func(t *testing.T) {
		got := processEntriesForDisplay(entries, langs, language.Und, language.English, false)
		for i := 1; i < len(got); i++ {
			if got[i-1].Key > got[i].Key {
				t.Errorf("expected sorted keys, got %q before %q", got[i-1].Key, got[i].Key)
			}
		}
	})
}

func TestIsDependencyKeyForEntry(t *testing.T) {
	deps := map[string]bool{
		"com.github.some-dep": true,
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"com.github.some-dep.feature.key", true},
		{"com.github.some-dep", true},
		{"com.github.other.feature.key", false},
		{"app.own.key", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := isDependencyKeyForEntry(tt.key, deps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("isDependencyKeyForEntry(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
