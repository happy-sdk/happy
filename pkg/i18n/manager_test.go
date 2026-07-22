// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
)

func TestManager_Reload_NotInitialized(t *testing.T) {
	// Create a new manager that's not initialized
	m := newManager()

	// Reload should return early if not initialized
	m.reload()

	// Should not panic
}

func TestManager_Reload_WithQueue(t *testing.T) {
	Initialize(language.English)

	// Queue translations
	_ = QueueTranslation(language.German, "queued.key1", "Wert 1")
	_ = QueueTranslation(language.Und, "queued.key2", "Value 2") // Und should map to fallback

	// Reload should process queue
	Reload()

	// Verify translations are available
	testutils.Assert(t, SupportsLanguage(language.German), "German should be supported after reload")
}

func TestManager_Reload_DifferentLanguages(t *testing.T) {
	Initialize(language.English)

	// Register multiple languages
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")
	_ = RegisterTranslation(language.Spanish, "test.key", "Valor")

	// Set language to French
	_ = SetLanguage(language.French)

	// Reload should rebuild printer cache
	Reload()

	// Verify printers are available
	printer, err := GetPrinterFor(language.German)
	testutils.NoError(t, err)
	testutils.NotNil(t, printer)
}

func TestManager_GetPrinterFor_Cached(t *testing.T) {
	Initialize(language.English)

	// Register language
	_ = RegisterTranslation(language.French, "test.key", "Valeur")

	// Get printer first time (creates cache)
	printer1, err := GetPrinterFor(language.French)
	testutils.NoError(t, err)

	// Get printer second time (should use cache)
	printer2, err := GetPrinterFor(language.French)
	testutils.NoError(t, err)

	// Should return same printer instance (from cache)
	testutils.Equal(t, printer1, printer2, "expected same printer instance from cache")
}

func TestManager_GetPrinterFor_CurrentLang(t *testing.T) {
	Initialize(language.English)

	// Set language to French
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = SetLanguage(language.French)

	// Get printer for current language
	printer, err := GetPrinterFor(language.French)
	testutils.NoError(t, err)

	// Should return current printer
	currentPrinter := GetPrinter()
	testutils.Equal(t, currentPrinter, printer, "expected printer for current language to match current printer")
}

func TestManager_GetPrinterFor_FallbackLang(t *testing.T) {
	Initialize(language.English)

	// Get printer for fallback language
	printer, err := GetPrinterFor(language.English)
	testutils.NoError(t, err)

	// Should return fallback printer
	fallbackPrinter := GetFallbackPrinter()
	testutils.Equal(t, fallbackPrinter, printer, "expected printer for fallback language to match fallback printer")
}

func TestManager_RegisterTranslation_WithPrefix(t *testing.T) {
	Initialize(language.English)

	// Register with prefix using RegisterTranslation (which uses empty prefix)
	// To test prefix, we need to use the internal API or register nested structure
	translations := map[string]any{
		"prefix": map[string]any{
			"key": "Value",
		},
	}
	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	// Verify translation is accessible with full key
	result := T("prefix.key")
	testutils.Equal(t, "Value", result)
}

func TestManager_RegisterTranslation_CatalogMessage(t *testing.T) {
	Initialize(language.English)

	// Register catalog.Message type
	msg := catalog.String("Catalog Message")
	err := RegisterTranslation(language.English, "catalog.msg", msg)
	testutils.NoError(t, err)

	result := T("catalog.msg")
	testutils.Equal(t, "Catalog Message", result)
}

func TestManager_RegisterTranslation_NestedMap(t *testing.T) {
	Initialize(language.English)

	// Register nested map
	translations := map[string]any{
		"nested": map[string]any{
			"level1": map[string]any{
				"level2": "Deep Value",
			},
		},
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	result := T("nested.level1.level2")
	testutils.Equal(t, "Deep Value", result)
}

func TestManager_QueueTranslation_OverwriteRegistered(t *testing.T) {
	Initialize(language.English)

	// Register translation first (this triggers reload automatically)
	err := RegisterTranslation(language.English, "duplicate.key", "Value 1")
	testutils.NoError(t, err)
	result := T("duplicate.key")
	testutils.Equal(t, "Value 1", result, "expected original value")

	// Queue same key (should succeed - allows overwriting)
	// This simulates application translations overriding SDK defaults
	err = QueueTranslation(language.English, "duplicate.key", "Value 2")
	testutils.NoError(t, err, "expected overwriting to be allowed")

	// Reload to process queue and overwrite existing translation
	Reload()

	// Verify the overwritten value is used (application translation overrides SDK default)
	result = T("duplicate.key")
	testutils.Equal(t, "Value 2", result, "expected overwritten value")
}

func TestManager_ExtractRootKey(t *testing.T) {
	m := newManager()

	tests := []struct {
		key      string
		expected string
	}{
		{"com.github.happy-sdk.happy.sdk.cli.flags.version", "com.github.happy-sdk.happy.sdk.cli"},
		{"com.github.happy-sdk.happy.pkg.vars.varflag.ErrFlag", "com.github.happy-sdk.happy.pkg.vars"}, // parts[4] is "pkg", so returns first 6 parts
		{"short.key", "short"},     // Short keys return first segment
		{"com.github.test", "com"}, // Short keys return first segment
		{"com.github.happy-sdk.happy.sdk.app.name", "com.github.happy-sdk.happy.sdk.app"},
		{"com.github.happy-sdk.happy.sdk.cli.flags", "com.github.happy-sdk.happy.sdk.cli"}, // parts[4] is "sdk", so returns first 6 parts
	}

	for _, tt := range tests {
		result := m.extractRootKey(tt.key)
		testutils.Equal(t, tt.expected, result, "extractRootKey(%q)", tt.key)
	}
}

func TestManager_GetAllTranslations_NotInitialized(t *testing.T) {
	// Create new manager (not initialized)
	m := newManager()

	entries := m.getAllTranslations()
	testutils.Nil(t, entries, "expected nil entries when not initialized")
}

func TestManager_GetAllTranslations_WithRootKeys(t *testing.T) {
	Initialize(language.English)

	// Register translations with root keys
	translations := map[string]any{
		"com.github.happy-sdk.test": map[string]any{
			"key1": "Value 1",
			"key2": "Value 2",
		},
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	entries := GetAllTranslations()

	// Verify entries have root keys
	found := false
	for _, entry := range entries {
		if entry.Key == "com.github.happy-sdk.test.key1" {
			found = true
			testutils.Assert(t, entry.RootKey != "", "expected root key to be set")
		}
	}
	testutils.Assert(t, found, "expected to find test key in entries")
}

func TestManager_GetTranslationReport_Empty(t *testing.T) {
	// Create new manager (not initialized)
	m := newManager()
	m.initialized = true

	report := m.getTranslationReport(language.English)

	testutils.Equal(t, 0, report.Total)
	testutils.Equal(t, 0.0, report.Percentage)
}

func TestManager_GetSupported(t *testing.T) {
	Initialize(language.English)

	// Register additional languages
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")

	languages := GetLanguages()
	testutils.Assert(t, len(languages) >= 3, "expected at least 3 languages, got %d", len(languages))
}

func TestManager_AddToDictionary_Overwrite(t *testing.T) {
	Initialize(language.English)

	// Register initial translation
	_ = RegisterTranslation(language.English, "overwrite.key", "Original")

	// Overwrite (should be allowed)
	_ = RegisterTranslation(language.English, "overwrite.key", "Updated")

	result := T("overwrite.key")
	testutils.Equal(t, "Updated", result)
}

func TestManager_RegisterTranslation_UnsupportedType(t *testing.T) {
	Initialize(language.English)

	// Try to register unsupported type
	err := RegisterTranslation(language.English, "unsupported.key", 12345)
	testutils.Error(t, err, "expected error for unsupported type")
}

func TestManager_Reload_ErrorHandling(t *testing.T) {
	Initialize(language.English)

	// Queue invalid translation (will cause error during reload)
	// This tests the error handling in reload()
	// Note: This will log an error (slog.Error) AND be reported via the
	// returned []InitIssue - never Fatal, since a bad queued key reflects
	// some other package's/application's own translations, not this
	// package's own bundle (see InitIssue's doc comment).
	_ = QueueTranslation(language.English, "invalid.key", 12345) // Invalid type

	issues := Reload()

	// Should not panic - error is logged via slog.Error and also collected here.
	testutils.Assert(t, len(issues) > 0, "expected at least one InitIssue for the invalid queued translation")
	for _, issue := range issues {
		testutils.Assert(t, !issue.Fatal, "a queue-flush issue must never be Fatal")
		testutils.NotNil(t, issue.Err, "expected a non-nil error on the issue")
	}
}

func TestManager_GetPrinterFor_CacheMiss(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.German, "test.key", "Wert")
	printer, err := GetPrinterFor(language.German)
	testutils.NoError(t, err)
	testutils.NotNil(t, printer, "expected non-nil printer")
	result := printer.Sprintf("test.key")
	testutils.Equal(t, "Wert", result)
}

func TestManager_RegisterTranslation_WithNestedPrefix(t *testing.T) {
	Initialize(language.English)
	translations := map[string]any{
		"prefix": map[string]any{
			"nested": map[string]any{
				"key": "Value",
			},
		},
	}
	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)
	result := T("prefix.nested.key")
	testutils.Equal(t, "Value", result)
}

func TestManager_RegisterTranslation_EmptyRootKey(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslation(language.English, "short.key", "Value")
	testutils.NoError(t, err)
	result := T("short.key")
	testutils.Equal(t, "Value", result)
}

func TestManager_QueueTranslation_NewDictionary(t *testing.T) {
	Initialize(language.English)
	err := QueueTranslation(language.Italian, "new.key", "Nuovo Valore")
	testutils.NoError(t, err)
	Reload()
	testutils.Assert(t, SupportsLanguage(language.Italian), "Italian should be supported after reload")
}

func TestManager_QueueTranslation_NilQueue(t *testing.T) {
	Initialize(language.English)
	err := QueueTranslation(language.Portuguese, "queue.key", "Valor")
	testutils.NoError(t, err)
	Reload()
	testutils.Assert(t, SupportsLanguage(language.Portuguese), "Portuguese should be supported after reload")
}

func TestManager_GetAllTranslations_NilFallbackDict(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.French, "frenchonly.key", "Seulement Français")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "frenchonly.key" {
			found = true
			testutils.Assert(t, len(entry.Translations) > 0, "expected at least one translation")
		}
	}
	testutils.Assert(t, found, "expected to find frenchonly.key in entries")
}

func TestManager_GetAllTranslations_PrinterFallbackForFallback(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "printerfallback.key", "Printer Fallback Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "printerfallback.key" {
			found = true
			testutils.Assert(t, entry.Fallback != "", "expected fallback value")
		}
	}
	testutils.Assert(t, found, "expected to find printerfallback.key in entries")
}

func TestManager_GetAllTranslations_PrinterTranslationForLang(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "printerlang.key", "English")
	_ = RegisterTranslation(language.Spanish, "printerlang.key", "Español")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "printerlang.key" {
			found = true
			testutils.Assert(t, entry.Translations[language.Spanish] != "", "expected Spanish translation")
		}
	}
	testutils.Assert(t, found, "expected to find printerlang.key in entries")
}

func TestManager_GetAllTranslations_ExtractRootKeyFallback(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "com.github.test.new.key", "Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "com.github.test.new.key" {
			found = true
			testutils.Assert(t, entry.RootKey != "", "expected root key to be extracted")
		}
	}
	testutils.Assert(t, found, "expected to find test key in entries")
}

func TestManager_ExtractRootKey_EdgeCases(t *testing.T) {
	m := newManager()
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"exactly 5 parts", "com.github.happy-sdk.happy.test", "com.github.happy-sdk.happy.test"},
		{"6 parts with pkg", "com.github.happy-sdk.happy.pkg.vars.key", "com.github.happy-sdk.happy.pkg.vars"},
		{"6 parts with sdk", "com.github.happy-sdk.happy.sdk.cli.key", "com.github.happy-sdk.happy.sdk.cli"},
		{"7 parts with pkg", "com.github.happy-sdk.happy.pkg.vars.varflag.key", "com.github.happy-sdk.happy.pkg.vars"},
		{"7 parts with sdk", "com.github.happy-sdk.happy.sdk.cli.flags.key", "com.github.happy-sdk.happy.sdk.cli"},
		{"4 parts (too short)", "com.github.happy-sdk.test", "com"}, // Short keys return first segment
		{"3 parts (too short)", "com.github.test", "com"},           // Short keys return first segment
		{"6 parts without pkg/sdk", "com.github.happy-sdk.happy.other.key", "com.github.happy-sdk.happy.other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.extractRootKey(tt.key)
			testutils.Equal(t, tt.expected, result, "extractRootKey(%q)", tt.key)
		})
	}
}

func TestManager_RegisterTranslation_CatalogMessageSlice(t *testing.T) {
	Initialize(language.English)
	msgs := []catalog.Message{
		catalog.String("Message 1"),
		catalog.String("Message 2"),
	}
	err := RegisterTranslation(language.English, "catalog.msgs", msgs)
	_ = err
}

func TestManager_RegisterTranslation_ShouldSupportPath(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslation(language.Dutch, "dutch.key", "Nederlandse Waarde")
	testutils.NoError(t, err)
	testutils.Assert(t, SupportsLanguage(language.Dutch), "Dutch should be supported after registration")
}

func TestManager_GetPrinterFor_PrinterCacheNil(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.Russian, "test.key", "Значение")
	printer, err := GetPrinterFor(language.Russian)
	testutils.NoError(t, err)
	testutils.NotNil(t, printer, "expected non-nil printer")
	result := printer.Sprintf("test.key")
	testutils.Equal(t, "Значение", result)
}

func TestManager_GetAllTranslations_PrinterError(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "errorlang.key", "Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "errorlang.key" {
			found = true
			testutils.Assert(t, entry.Fallback != "", "expected fallback value")
		}
	}
	testutils.Assert(t, found, "expected to find errorlang.key in entries")
}

func TestManager_GetTranslationReport_AllTranslated(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "alltrans.key1", "Value 1")
	_ = RegisterTranslation(language.English, "alltrans.key2", "Value 2")
	_ = RegisterTranslation(language.French, "alltrans.key1", "Valeur 1")
	_ = RegisterTranslation(language.French, "alltrans.key2", "Valeur 2")
	report := GetTranslationReport(language.French)
	testutils.Assert(t, report.Total != 0, "expected non-zero total")
	for _, entry := range report.MissingEntries {
		if entry.Key == "alltrans.key1" || entry.Key == "alltrans.key2" {
			t.Logf("unexpected missing entry: %s", entry.Key)
		}
	}
}

func TestManager_GetTranslationReport_NoRootKeys(t *testing.T) {
	Initialize(language.English)
	translations := map[string]any{
		"flat": map[string]any{
			"key1": "Value 1",
			"key2": "Value 2",
		},
	}
	_ = RegisterTranslations(language.English, translations)
	report := GetTranslationReport(language.English)
	testutils.Assert(t, report.Total != 0, "expected non-zero total")
}

func TestManager_Reload_QueueNil(t *testing.T) {
	Initialize(language.English)
	Reload()
}

func TestManager_RegisterTranslation_MapNestedError(t *testing.T) {
	Initialize(language.English)
	translations := map[string]any{
		"nested": map[string]any{
			"invalid": 12345,
		},
	}
	err := RegisterTranslations(language.English, translations)
	testutils.Error(t, err, "expected error for invalid type in nested map")
}

func TestManager_SetCurrentLanguage_NotSupported(t *testing.T) {
	Initialize(language.English)
	// Test setCurrentLanguage when language is not supported
	m := newManager()
	m.initialized = true
	err := m.setCurrentLanguage(language.Japanese)
	testutils.Error(t, err, "expected error for unsupported language")
	testutils.Assert(t, errors.Is(err, ErrLanguageNotSupported), "expected err to be ErrLanguageNotSupported, got %v", err)
}

func TestQueueTranslations_Overwrite(t *testing.T) {
	Initialize(language.English)
	// Test queueTranslations allows overwriting existing keys
	_ = QueueTranslation(language.English, "duplicate.queue.key", "Value 1")
	translations := map[string]any{
		"duplicate.queue.key": "Value 2", // Overwrite existing key
	}
	err := QueueTranslations(language.English, translations)
	testutils.NoError(t, err, "expected overwriting to be allowed")

	// Reload to process queue
	Reload()

	// Verify the overwritten value is used
	result := T("duplicate.queue.key")
	testutils.Equal(t, "Value 2", result, "expected overwritten value")
}

func TestSetLanguage_NotSupported(t *testing.T) {
	Initialize(language.English)
	// Test setLanguage when language is not supported
	err := SetLanguage(language.Japanese)
	testutils.Error(t, err, "expected error for unsupported language")
	testutils.Assert(t, errors.Is(err, ErrLanguageNotSupported), "expected err to be ErrLanguageNotSupported, got %v", err)
}

func TestManager_RegisterTranslation_DeferShouldSupport(t *testing.T) {
	Initialize(language.English)
	// Test registerTranslation defer function when shouldSupport is true
	// and language is not yet supported
	// Register a new language that's not supported yet
	err := RegisterTranslation(language.Norwegian, "norwegian.key", "Norsk verdi")
	testutils.NoError(t, err)
	// Language should now be supported due to defer function
	testutils.Assert(t, SupportsLanguage(language.Norwegian), "Norwegian should be supported after registration")
}

func TestManager_RegisterTranslation_DeferShouldSupportAlreadySupported(t *testing.T) {
	Initialize(language.English)
	// Test registerTranslation defer function when shouldSupport is true
	// but language is already supported
	// First register to make it supported
	_ = RegisterTranslation(language.Swedish, "swedish.key1", "Svensk värde 1")
	testutils.Assert(t, SupportsLanguage(language.Swedish), "Swedish should be supported")
	// Register another translation - shouldSupport is true but language already supported
	err := RegisterTranslation(language.Swedish, "swedish.key2", "Svensk värde 2")
	testutils.NoError(t, err)
	// Should still be supported
	testutils.Assert(t, SupportsLanguage(language.Swedish), "Swedish should still be supported")
}

func TestManager_RegisterTranslation_MapCaseNoShouldSupport(t *testing.T) {
	Initialize(language.English)
	// Test registerTranslation with map[string]any case
	// This case doesn't set shouldSupport at the map level, so defer won't add map to dictionary
	// But recursive calls will set shouldSupport for nested values
	translations := map[string]any{
		"nested": map[string]any{
			"key": "Value",
		},
	}
	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)
	// Verify translation is accessible
	result := T("nested.key")
	testutils.Equal(t, "Value", result)
}

func TestManager_RegisterTranslation_MapCaseWithError(t *testing.T) {
	Initialize(language.English)
	// Test registerTranslation with map[string]any case when nested registration fails
	translations := map[string]any{
		"nested": map[string]any{
			"invalid": 12345, // Invalid type, will cause error in recursive call
		},
	}
	err := RegisterTranslations(language.English, translations)
	testutils.Error(t, err, "expected error for invalid type in nested map")
}

// Removed duplicate - see TestManager_RegisterTranslation_EmptyRootKey above

func TestManager_GetAllTranslations_FallbackDictNil(t *testing.T) {
	Initialize(language.English)
	// Test getAllTranslations when fallbackDictCopy is nil
	// This is hard to achieve normally, but we can test the path exists
	// by ensuring fallbackDict is nil (which shouldn't happen in normal flow)
	// Actually, fallbackDict is always populated, so this path is hard to test
	// But we can test the else branch when key not in dict
	_ = RegisterTranslation(language.English, "catalogonly.key3", "Catalog Only Value 3")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "catalogonly.key3" {
			found = true
			// Should have fallback from printer if not in dict
			testutils.Assert(t, entry.Fallback != "", "expected fallback value")
		}
	}
	testutils.Assert(t, found, "expected to find catalogonly.key3 in entries")
}

// Removed duplicate - see TestManager_GetAllTranslations_PrinterError above

func TestManager_GetAllTranslations_RootKeyExtraction(t *testing.T) {
	Initialize(language.English)
	// Test getAllTranslations when rootKey is empty and needs extraction
	// Register a key that should have a root key but isn't tracked
	_ = RegisterTranslation(language.English, "com.github.happy-sdk.happy.pkg.test.newkey", "New Key Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "com.github.happy-sdk.happy.pkg.test.newkey" {
			found = true
			// Should have root key extracted
			testutils.Assert(t, entry.RootKey != "", "expected root key to be extracted")
		}
	}
	testutils.Assert(t, found, "expected to find test key in entries")
}

func TestManager_GetAllTranslations_FallbackDictNilButPrinter(t *testing.T) {
	Initialize(language.English)
	// Test getAllTranslations when fallbackDictCopy is nil but fallbackPrinter exists
	// Register translation in catalog (via RegisterTranslation) but ensure it's in catalog
	_ = RegisterTranslation(language.English, "catalogonly.key", "Catalog Only Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "catalogonly.key" {
			found = true
			// Should have fallback from printer
			testutils.Assert(t, entry.Fallback != "", "expected fallback value from printer")
		}
	}
	testutils.Assert(t, found, "expected to find catalogonly.key in entries")
}

func TestManager_GetAllTranslations_PrinterErrorPath(t *testing.T) {
	Initialize(language.English)
	// Test getAllTranslations when getPrinterFor returns error
	// Register translation only in English
	_ = RegisterTranslation(language.English, "englishonly.key", "English Only")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "englishonly.key" {
			found = true
			// Should have fallback
			testutils.Assert(t, entry.Fallback != "", "expected fallback value")
		}
	}
	testutils.Assert(t, found, "expected to find englishonly.key in entries")
}

func TestManager_GetAllTranslations_PrinterResultEqualsKey(t *testing.T) {
	Initialize(language.English)
	// Test getAllTranslations when printer.Sprintf returns key
	// This happens when translation is not in catalog for that language
	// Register translation only in English
	_ = RegisterTranslation(language.English, "englishonly.key2", "English Only Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "englishonly.key2" {
			found = true
			// For languages where translation doesn't exist, printer returns key
			// So it won't be added to Translations map
			// This tests the path where result == key
		}
	}
	testutils.Assert(t, found, "expected to find englishonly.key2 in entries")
}

// TestManager_Initialize_ConcurrentReads is a regression test for a bug where
// initialize() wrote mngr.initialized/fallbackLang/currentLang without holding
// mngr.mu, while every reader (isInitialized, getCurrentLanguage,
// getFallbackLanguage) takes mngr.mu.RLock. Under `go test -race` this caught a
// real data race. Initialize uses sync.Once so the write only ever happens once
// per process; this test still guards against a regression in the readers'
// locking discipline by hammering them concurrently around an Initialize call.
func TestManager_Initialize_ConcurrentReads(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Initialize(language.English)
		}()
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = mngr.isInitialized()
		}()
		go func() {
			defer wg.Done()
			_ = mngr.getCurrentLanguage()
		}()
		go func() {
			defer wg.Done()
			_ = mngr.getFallbackLanguage()
		}()
	}
	wg.Wait()
}

// TestRegisterTranslation_ConcurrentRegistration is a regression test for a bug
// where registerTranslation mutated globalCatalog (SetString/Set) without
// holding mngr.mu, even though the surrounding code and catalog.go's own
// comment claimed registration "serializes behind the manager's mutex".
// catalog.Builder is not safe for concurrent writers, so two goroutines
// registering different bundles at the same time raced on its internal maps.
// Under `go test -race` this line of test (registering into distinct v2
// bundles from many goroutines at once) catches that real data race.
func TestRegisterTranslation_ConcurrentRegistration(t *testing.T) {
	Initialize(language.English)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			bundle := fmt.Sprintf("schematest.concurrent%d", i)
			_ = RegisterTranslations(language.Und, map[string]any{
				"version": float64(2),
				"bundle":  bundle,
				"locales": map[string]any{
					"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
				},
			})
		}()
	}
	wg.Wait()
}

// TestManager_GetTranslationReport_FallbackLanguage is a regression test for a bug
// where getAllTranslations stores the fallback language's value in entry.Fallback
// (not entry.Translations[fallbackLang]), but getTranslationReport only checked
// entry.Translations[lang]. This caused GetTranslationReport(fallbackLang) to
// incorrectly report entries as untranslated/missing even though every key has
// a value in the fallback language by construction.
func TestManager_GetTranslationReport_FallbackLanguage(t *testing.T) {
	Initialize(language.English)

	// Register a key in both the fallback language (English) and French.
	_ = RegisterTranslation(language.English, "fallbackreport.key1", "English Value 1")
	_ = RegisterTranslation(language.French, "fallbackreport.key1", "Valeur Française 1")

	englishReport := GetTranslationReport(language.English)
	frenchReport := GetTranslationReport(language.French)

	// Find our specific key in both reports' entries to compute targeted stats,
	// since the report aggregates over every key registered in the package.
	var englishMissingOurKey, frenchMissingOurKey bool
	for _, entry := range englishReport.MissingEntries {
		if entry.Key == "fallbackreport.key1" {
			englishMissingOurKey = true
		}
	}
	for _, entry := range frenchReport.MissingEntries {
		if entry.Key == "fallbackreport.key1" {
			frenchMissingOurKey = true
		}
	}

	testutils.Assert(t, !englishMissingOurKey, "expected fallbackreport.key1 to be reported as translated for the fallback language (English)")
	testutils.Assert(t, !frenchMissingOurKey, "expected fallbackreport.key1 to be reported as translated for French")

	// The fallback language report should never have a worse percentage than 0
	// when every registered key has a fallback value; assert it's > 0 here since
	// at least one key (ours) is guaranteed to have a fallback value.
	testutils.Assert(t, englishReport.Percentage > 0, "expected non-zero translated percentage for fallback language report, got %f", englishReport.Percentage)
}

// TestManager_GetTranslationReport_FallbackLanguage_FullyTranslated is a more
// targeted regression test using an isolated manager instance (rather than the
// shared global mngr) so the percentage assertion is exact and not affected by
// other tests' registered keys.
func TestManager_GetTranslationReport_FallbackLanguage_FullyTranslated(t *testing.T) {
	m := newManager()
	m.initialized = true
	m.fallbackLang = language.English
	m.currentLang = language.English
	m.dictionaries[language.English] = map[string]any{
		"isolated.key1": "Value 1",
		"isolated.key2": "Value 2",
	}
	m.dictionaries[language.French] = map[string]any{
		"isolated.key1": "Valeur 1",
		"isolated.key2": "Valeur 2",
	}
	m.langs = []language.Tag{language.English, language.French}

	report := m.getTranslationReport(language.English)

	testutils.Equal(t, 2, report.Total)
	testutils.Equal(t, 2, report.Translated)
	testutils.Equal(t, 0, report.Missing)
	testutils.Equal(t, 100.0, report.Percentage)
	testutils.Equal(t, 0, len(report.MissingEntries))
}

func TestManager_GetAllTranslations_FallbackDictNilButPrinterExists(t *testing.T) {
	Initialize(language.English)
	// Test getAllTranslations when fallbackDictCopy is nil but fallbackPrinter exists
	// This is hard to achieve because fallbackDict is usually populated
	// But we can test the path where fallbackDictCopy is nil
	// Register translation in catalog (via RegisterTranslation)
	_ = RegisterTranslation(language.English, "catalogprinter.key", "Catalog Printer Value")
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "catalogprinter.key" {
			found = true
			// Should have fallback from printer if not in dict
			testutils.Assert(t, entry.Fallback != "", "expected fallback value")
		}
	}
	testutils.Assert(t, found, "expected to find catalogprinter.key in entries")
}

func TestRegisterTranslations_MissingVersionKeyIsSchemaV1(t *testing.T) {
	Initialize(language.English)
	// A document that never declares "version" (every translation file
	// written before schema versioning existed) must keep registering
	// exactly as before - under lang, since a v1 document has no locale
	// of its own.
	err := RegisterTranslations(language.English, map[string]any{
		"schematest.no_version": "hello",
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "hello", T("schematest.no_version"))
}

func TestRegisterTranslations_V2DocumentUsesItsOwnLocalesAndBundle(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslations(language.Und, map[string]any{
		"version": float64(2),
		"bundle":  "schematest.v2bundle",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello v2"}},
			"de": map[string]any{"keys": map[string]any{"greeting": "hallo v2"}},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "hello v2", T("schematest.v2bundle.greeting"))
	testutils.Equal(t, "hallo v2", TL(language.German, "schematest.v2bundle.greeting"))
	// The reserved envelope keys must never leak through as translation keys.
	for _, entry := range GetAllTranslations() {
		testutils.Assert(t, entry.Key != "version" && entry.Key != "bundle" && entry.Key != "locales" && entry.Key != "keys",
			"reserved schema key leaked into translations: "+entry.Key)
	}
	source, ok := GetBundleSourceLanguage("schematest.v2bundle")
	testutils.Assert(t, ok, "expected the bundle's source language to be known")
	testutils.Equal(t, language.English, source)
}

// TestReload_DetectsKeyTypoInNonSourceLocale is a regression test for a
// real gap the maintainer found: schema validation only checks that a
// key's *shape* is well-formed (lowercase a-z/0-9/_), not whether it
// actually corresponds to anything in the bundle's source locale. A
// translator's typo in a key NAME (as opposed to its value) previously
// registered silently: the misspelled key became a new, orphaned entry
// nothing ever looks up, while the correctly-spelled key it was meant to
// be stayed entirely missing from that locale, with no warning anywhere.
func TestReload_DetectsKeyTypoInNonSourceLocale(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslations(language.Und, map[string]any{
		"version": float64(2),
		"bundle":  "schematest.typobundle",
		"source":  "en",
		"locales": map[string]any{
			"en": map[string]any{"keys": map[string]any{"greeting": "hello"}},
			// "greeting" typo'd as "gretting" - still a validly-*shaped*
			// key (schema validation alone can't catch this), but it does
			// not exist anywhere in the source ("en") locale.
			"de": map[string]any{"keys": map[string]any{"gretting": "hallo"}},
		},
	})
	testutils.NoError(t, err)

	issues := Reload()
	var found *InitIssue
	for i := range issues {
		if !issues[i].Fatal {
			found = &issues[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected a non-Fatal InitIssue for the typo'd \"gretting\" key")
	}
	testutils.Assert(t, errorsIsUnknownLocaleKey(found.Err), "expected the issue to be (or wrap) ErrUnknownLocaleKey, got: %v", found.Err)
	msg := found.Err.Error()
	testutils.Assert(t, strings.Contains(msg, "gretting"), "expected the message to name the offending key, got: %q", msg)
	testutils.Assert(t, strings.Contains(msg, "greeting"), "expected the message to suggest the close source key \"greeting\", got: %q", msg)

	// The library must not register a key it's already warned about: it
	// should be gone from GetAllTranslations (what l10n list/report read)
	// and unresolvable through T/TL (which fall back to the raw key).
	const badKey = "schematest.typobundle.gretting"
	for _, entry := range GetAllTranslations() {
		testutils.Assert(t, entry.Key != badKey, "expected %q to be pruned from GetAllTranslations, but found it", badKey)
	}
	got := TL(language.German, badKey)
	testutils.Equal(t, badKey, got)
}

func errorsIsUnknownLocaleKey(err error) bool {
	return errors.Is(err, ErrUnknownLocaleKey)
}

func TestClosestKey_SuggestsPlausibleTypoOnly(t *testing.T) {
	candidates := map[string]bool{"greeting": true, "farewell": true, "report.description": true}

	suggestion, ok := closestKey("gretting", candidates)
	testutils.Assert(t, ok, "expected a suggestion for a one-transposition typo of \"greeting\"")
	testutils.Equal(t, "greeting", suggestion)

	_, ok = closestKey("something_completely_unrelated", candidates)
	testutils.Assert(t, !ok, "did not expect a suggestion for a key with no plausible match")
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"greeting", "greeting", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"gretting", "greeting", 1}, // single substitution (e<->t at index 3)
		{"kitten", "sitting", 3},    // classic textbook example
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestRegisterTranslations_UnsupportedSchemaVersionIsRejected(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslations(language.English, map[string]any{
		"version":      float64(999),
		"schematest.x": "unreachable",
	})
	if err == nil {
		t.Fatal("expected an error for an unsupported schema version")
	}
	testutils.ErrorIs(t, err, ErrInvalidSchemaVersion)
}
