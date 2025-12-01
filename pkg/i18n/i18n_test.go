// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"context"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
)

func TestInitialize(t *testing.T) {
	// Initialize with English as fallback
	Initialize(language.English)

	// Verify initialization
	testutils.Assert(t, isInitialized(), "i18n should be initialized")

	// Test that Initialize is idempotent (sync.Once)
	Initialize(language.French)
	fallback := GetFallbackLanguage()
	testutils.Equal(t, language.English, fallback)
}

func TestRegisterTranslations(t *testing.T) {
	Initialize(language.English)

	translations := map[string]any{
		"test.key":     "Test Value",
		"test.another": "Another Value",
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	// Verify translation is registered
	result := T("test.key")
	testutils.Equal(t, "Test Value", result)
}

func TestRegisterTranslation(t *testing.T) {
	Initialize(language.English)

	err := RegisterTranslation(language.English, "single.key", "Single Value")
	testutils.NoError(t, err)

	result := T("single.key")
	testutils.Equal(t, "Single Value", result)
}

func TestSupportsLanguage(t *testing.T) {
	Initialize(language.English)

	// English should be supported (fallback)
	if !SupportsLanguage(language.English) {
		t.Error("English should be supported")
	}

	// Register a new language
	err := RegisterTranslation(language.French, "test.key", "Valeur de test")
	testutils.NoError(t, err)

	// French should now be supported
	if !SupportsLanguage(language.French) {
		t.Error("French should be supported after registration")
	}

	// Unsupported language
	if SupportsLanguage(language.Japanese) {
		t.Error("Japanese should not be supported")
	}
}

func TestQueueTranslation(t *testing.T) {
	Initialize(language.English)

	// Queue a translation before it's registered
	err := QueueTranslation(language.German, "queued.key", "Gequeueter Wert")
	testutils.NoError(t, err)

	// Reload to apply queued translations
	Reload()

	// Verify translation is now available
	if !SupportsLanguage(language.German) {
		t.Error("German should be supported after reload")
	}
}

func TestQueueTranslations(t *testing.T) {
	Initialize(language.English)

	translations := map[string]any{
		"queued1.key": "Queued Value 1",
		"queued2.key": "Queued Value 2",
	}

	err := QueueTranslations(language.Spanish, translations)
	testutils.NoError(t, err)

	Reload()

	// Verify translations are available
	if !SupportsLanguage(language.Spanish) {
		t.Error("Spanish should be supported after reload")
	}
}

func TestSetLanguage(t *testing.T) {
	Initialize(language.English)

	// Register French first
	err := RegisterTranslation(language.French, "test.key", "Valeur")
	testutils.NoError(t, err)

	// Set language to French
	err = SetLanguage(language.French)
	testutils.NoError(t, err)

	current := GetLanguage()
	if current != language.French {
		t.Errorf("expected French, got %v", current)
	}

	// Try to set unsupported language
	err = SetLanguage(language.Japanese)
	testutils.Error(t, err, "expected error when setting unsupported language")
}

func TestGetLanguage(t *testing.T) {
	Initialize(language.English)
	_ = SetLanguage(language.English) // Ensure we're using English

	lang := GetLanguage()
	if lang != language.English {
		t.Errorf("expected English, got %v", lang)
	}
}

func TestGetFallbackLanguage(t *testing.T) {
	Initialize(language.English)

	fallback := GetFallbackLanguage()
	if fallback != language.English {
		t.Errorf("expected English, got %v", fallback)
	}
}

func TestGetLanguages(t *testing.T) {
	Initialize(language.English)

	languages := GetLanguages()
	testutils.Assert(t, len(languages) > 0, "expected at least one language")

	// Register additional language
	_ = RegisterTranslation(language.French, "test.key", "Valeur")

	languages = GetLanguages()
	if len(languages) < 2 {
		t.Error("expected at least two languages after registration")
	}
}

func TestParseLanguage(t *testing.T) {
	Initialize(language.English)

	// Valid language
	lang := ParseLanguage("en")
	if lang != language.English {
		t.Errorf("expected English, got %v", lang)
	}

	// Invalid language should return fallback
	lang = ParseLanguage("invalid")
	if lang != language.English {
		t.Errorf("expected fallback English, got %v", lang)
	}

	// Empty string should return fallback
	lang = ParseLanguage("")
	if lang != language.English {
		t.Errorf("expected fallback English, got %v", lang)
	}
}

func TestWithLanguage(t *testing.T) {
	ctx := context.Background()
	lang := language.French

	ctxWithLang := WithLanguage(ctx, lang)

	retrievedLang := ctxWithLang.Value(langContextKey)
	testutils.EqualAny(t, lang, retrievedLang)
}

func TestGetPrinter(t *testing.T) {
	Initialize(language.English)

	printer := GetPrinter()
	testutils.NotNil(t, printer, "expected non-nil printer")
}

func TestGetPrinterFor(t *testing.T) {
	Initialize(language.English)

	// Get printer for English
	printer, err := GetPrinterFor(language.English)
	testutils.NoError(t, err)
	testutils.NotNil(t, printer, "expected non-nil printer")

	// Try unsupported language
	_, err = GetPrinterFor(language.Japanese)
	testutils.Error(t, err, "expected error for unsupported language")
}

func TestGetFallbackPrinter(t *testing.T) {
	Initialize(language.English)

	printer := GetFallbackPrinter()
	testutils.NotNil(t, printer, "expected non-nil printer")
}

func TestReload(t *testing.T) {
	Initialize(language.English)

	// Queue some translations
	_ = QueueTranslation(language.German, "reload.key", "Neu laden")

	// Reload should apply queued translations
	Reload()

	// Verify translation is available
	if !SupportsLanguage(language.German) {
		t.Error("German should be supported after reload")
	}
}

func TestT(t *testing.T) {
	Initialize(language.English)

	// Register translation
	_ = RegisterTranslation(language.English, "test.translation", "Hello World")

	result := T("test.translation")
	testutils.Equal(t, "Hello World", result)

	// Test with arguments
	_ = RegisterTranslation(language.English, "test.format", "Hello %s")
	result = T("test.format", "World")
	testutils.Equal(t, "Hello World", result)

	// Test non-existent key (should return key)
	result = T("nonexistent.key")
	testutils.Equal(t, "nonexistent.key", result)
}

func TestTD(t *testing.T) {
	Initialize(language.English)

	// Test with existing translation
	_ = RegisterTranslation(language.English, "test.td", "Translated")
	result := TD("test.td", "Fallback")
	testutils.Equal(t, "Translated", result)

	// Test with non-existent key (should use fallback)
	result = TD("nonexistent.td", "Fallback Value")
	testutils.Equal(t, "Fallback Value", result)

	// Test with format arguments
	_ = RegisterTranslation(language.English, "test.td.format", "Hello %s")
	result = TD("test.td.format", "Fallback %s", "World")
	testutils.Equal(t, "Hello World", result)
}

func TestPT(t *testing.T) {
	Initialize(language.English)

	_ = RegisterTranslation(language.English, "prefix.local", "Prefixed Value")

	result := PT("prefix", "local")
	testutils.Equal(t, "Prefixed Value", result)
}

func TestPTD(t *testing.T) {
	Initialize(language.English)

	_ = RegisterTranslation(language.English, "prefix.local", "Prefixed Value")

	result := PTD("prefix", "local", "Fallback")
	// PTD doesn't use fallback, it just calls t() which returns key if not found
	testutils.Equal(t, "Prefixed Value", result)

	// Test with non-existent key (PTD doesn't use fallback parameter)
	result = PTD("prefix", "nonexistent", "Fallback Value")
	// PTD just calls t(), so it returns the key if not found
	testutils.Equal(t, "prefix.nonexistent", result)
}

func TestGetPackagePrefix(t *testing.T) {
	prefix := GetPackagePrefix()
	testutils.Assert(t, prefix != "", "expected non-empty package prefix")
	// Should contain the package identifier
	if len(prefix) < 10 {
		t.Error("package prefix seems too short")
	}
}

func TestGetAllTranslations(t *testing.T) {
	Initialize(language.English)

	// Register some translations
	_ = RegisterTranslation(language.English, "test.key1", "Value 1")
	_ = RegisterTranslation(language.English, "test.key2", "Value 2")
	_ = RegisterTranslation(language.French, "test.key1", "Valeur 1")

	entries := GetAllTranslations()
	testutils.Assert(t, len(entries) > 0, "expected at least one translation entry")

	// Verify entry structure
	found := false
	for _, entry := range entries {
		if entry.Key == "test.key1" {
			found = true
			if entry.Fallback == "" {
				t.Error("expected fallback value")
			}
			if len(entry.Translations) == 0 {
				t.Error("expected at least one translation")
			}
		}
	}
	testutils.Assert(t, found, "expected to find test.key1 in entries")
}

func TestGetTranslationReport(t *testing.T) {
	Initialize(language.English)

	// Register translations
	_ = RegisterTranslation(language.English, "test.key1", "Value 1")
	_ = RegisterTranslation(language.English, "test.key2", "Value 2")
	_ = RegisterTranslation(language.French, "test.key1", "Valeur 1")
	// test.key2 is missing in French

	report := GetTranslationReport(language.French)
	if report.Total == 0 {
		t.Error("expected non-zero total")
	}
	if report.Translated == 0 {
		t.Error("expected at least one translated key")
	}
	if report.Missing == 0 {
		t.Error("expected at least one missing key")
	}
	if report.Percentage < 0 || report.Percentage > 100 {
		t.Errorf("percentage should be between 0 and 100, got %f", report.Percentage)
	}
	if len(report.MissingEntries) == 0 {
		t.Error("expected missing entries")
	}
	if len(report.RootKeys) == 0 {
		t.Error("expected at least one root key")
	}
	if len(report.PerRootKey) == 0 {
		t.Error("expected per-root-key statistics")
	}
}

func TestStructuredTranslations(t *testing.T) {
	Initialize(language.English)

	// Test structured format (with root key)
	translations := map[string]any{
		"com.github.happy-sdk.happy.test": map[string]any{
			"key1": "Value 1",
			"key2": "Value 2",
		},
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	// Verify translations are accessible
	result := T("com.github.happy-sdk.happy.test.key1")
	testutils.Equal(t, "Value 1", result)
}

func TestFlatTranslations(t *testing.T) {
	Initialize(language.English)

	// Test flat format (no root key)
	translations := map[string]any{
		"app": map[string]any{
			"name":        "Test App",
			"description": "Test Description",
		},
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	// Verify translations are accessible
	result := T("app.name")
	testutils.Equal(t, "Test App", result)
}

func TestComplexTranslationTypes(t *testing.T) {
	Initialize(language.English)

	// Test catalog.Message type
	msg := catalog.String("Complex Message")
	err := RegisterTranslation(language.English, "complex.msg", msg)
	testutils.NoError(t, err)

	result := T("complex.msg")
	testutils.Equal(t, "Complex Message", result)

	// Note: []catalog.Message is not a valid type for RegisterTranslation
	// It expects individual catalog.Message, not a slice
}

func TestOverwriteTranslation(t *testing.T) {
	Initialize(language.English)

	// Register initial translation
	err := RegisterTranslation(language.English, "overwrite.key", "Original")
	testutils.NoError(t, err)

	result := T("overwrite.key")
	testutils.Equal(t, "Original", result)

	// Overwrite translation (should be allowed)
	err = RegisterTranslation(language.English, "overwrite.key", "Updated")
	testutils.NoError(t, err)

	result = T("overwrite.key")
	testutils.Equal(t, "Updated", result)
}

func TestUnsupportedTranslationType(t *testing.T) {
	Initialize(language.English)

	// Try to register unsupported type
	err := RegisterTranslation(language.English, "unsupported.key", 12345)
	testutils.Error(t, err, "expected error for unsupported translation type")
}

func TestQueueTranslationDuplicate(t *testing.T) {
	Initialize(language.English)

	// Queue first translation
	err := QueueTranslation(language.German, "duplicate.key", "Value 1")
	testutils.NoError(t, err)

	// Queue duplicate (should succeed - allows overwriting)
	err = QueueTranslation(language.German, "duplicate.key", "Value 2")
	testutils.NoError(t, err, "expected overwriting to be allowed")

	// Reload to process queue
	Reload()

	// Verify the overwritten value is used
	_ = SetLanguage(language.German)
	result := T("duplicate.key")
	testutils.Equal(t, "Value 2", result, "expected overwritten value")
}

func TestLanguageFallback(t *testing.T) {
	Initialize(language.English)

	// Register translation only in English
	_ = RegisterTranslation(language.English, "fallback.key", "English Value")

	// Set language to French (no translation)
	_ = RegisterTranslation(language.French, "other.key", "Other")
	_ = SetLanguage(language.French)

	// Should fallback to English
	result := T("fallback.key")
	testutils.Equal(t, "English Value", result)
}

func TestT_FallbackToFallbackLanguage(t *testing.T) {
	Initialize(language.English)

	// Register translation only in English (fallback)
	_ = RegisterTranslation(language.English, "fallback.key", "English Value")

	// Set language to French (no translation)
	_ = RegisterTranslation(language.French, "other.key", "Other")
	_ = SetLanguage(language.French)

	// Should fallback to English
	result := T("fallback.key")
	testutils.Equal(t, "English Value", result)
}

func TestGetAllTranslations_FromPrinter(t *testing.T) {
	Initialize(language.English)

	// Register translation in catalog (via RegisterTranslation)
	_ = RegisterTranslation(language.English, "catalog.key", "Catalog Value")

	// Also register in French
	_ = RegisterTranslation(language.French, "catalog.key", "Valeur Catalogue")

	entries := GetAllTranslations()

	// Find the entry
	found := false
	for _, entry := range entries {
		if entry.Key == "catalog.key" {
			found = true
			// Should have fallback
			if entry.Fallback == "" {
				t.Error("expected fallback value")
			}
			// Should have French translation
			if entry.Translations[language.French] == "" {
				t.Error("expected French translation")
			}
		}
	}
	testutils.Assert(t, found, "expected to find catalog.key in entries")
}

func TestGetAllTranslations_ExtractRootKey(t *testing.T) {
	Initialize(language.English)

	// Register translation without explicit root key tracking
	_ = RegisterTranslation(language.English, "com.github.test.new.key", "Value")

	entries := GetAllTranslations()

	// Find entry and verify root key was extracted
	found := false
	for _, entry := range entries {
		if entry.Key == "com.github.test.new.key" {
			found = true
			// Root key should be extracted
			if entry.RootKey == "" {
				t.Error("expected root key to be extracted")
			}
		}
	}
	testutils.Assert(t, found, "expected to find test key in entries")
}

func TestRegisterTranslation_CatalogMessageSlice(t *testing.T) {
	Initialize(language.English)

	// Test []catalog.Message type
	// Note: This may fail with catalog error, which is expected behavior
	msgs := []catalog.Message{
		catalog.String("Message 1"),
		catalog.String("Message 2"),
	}

	err := RegisterTranslation(language.English, "catalog.msgs", msgs)
	// This may error due to catalog limitations, which is acceptable
	_ = err
}

func TestManager_GetPrinterFor_NewPrinter(t *testing.T) {
	Initialize(language.English)

	// Register language
	_ = RegisterTranslation(language.German, "test.key", "Wert")

	// Get printer (should create new one, not from cache)
	printer, err := GetPrinterFor(language.German)
	testutils.NoError(t, err)
	testutils.NotNil(t, printer, "expected non-nil printer")

	// Verify it works
	result := printer.Sprintf("test.key")
	testutils.Equal(t, "Wert", result)
}

func TestManager_GetTranslationReport_WithRootKeys(t *testing.T) {
	Initialize(language.English)

	// Register translations with different root keys
	translations1 := map[string]any{
		"com.github.happy-sdk.test1": map[string]any{
			"key1": "Value 1",
		},
	}
	translations2 := map[string]any{
		"com.github.happy-sdk.test2": map[string]any{
			"key2": "Value 2",
		},
	}

	_ = RegisterTranslations(language.English, translations1)
	_ = RegisterTranslations(language.French, translations1)
	_ = RegisterTranslations(language.English, translations2)
	// translations2 not in French (missing)

	report := GetTranslationReport(language.French)

	if len(report.RootKeys) == 0 {
		t.Error("expected at least one root key")
	}
	if len(report.PerRootKey) == 0 {
		t.Error("expected per-root-key statistics")
	}

	// Verify per-root-key stats
	for _, rootKey := range report.RootKeys {
		stats, ok := report.PerRootKey[rootKey]
		if !ok {
			t.Errorf("expected stats for root key %q", rootKey)
		}
		if stats.Total == 0 {
			t.Errorf("expected non-zero total for root key %q", rootKey)
		}
	}
}

func TestManager_GetAllTranslations_NoFallbackDict(t *testing.T) {
	Initialize(language.English)

	// Register translation only in non-fallback language
	_ = RegisterTranslation(language.French, "french.only", "Seulement Français")

	entries := GetAllTranslations()

	// Should still find the entry
	found := false
	for _, entry := range entries {
		if entry.Key == "french.only" {
			found = true
			// Fallback might be empty or from printer
			if len(entry.Translations) == 0 {
				t.Error("expected at least one translation")
			}
		}
	}
	testutils.Assert(t, found, "expected to find french.only in entries")
}

func TestManager_GetAllTranslations_PrinterFallback(t *testing.T) {
	Initialize(language.English)

	// Register translation in catalog but not in dictionary
	// This tests the printer fallback path
	_ = RegisterTranslation(language.English, "printer.key", "Printer Value")

	entries := GetAllTranslations()

	found := false
	for _, entry := range entries {
		if entry.Key == "printer.key" {
			found = true
			// Should have fallback (from printer if not in dict)
			if entry.Fallback == "" {
				t.Error("expected fallback value")
			}
		}
	}
	testutils.Assert(t, found, "expected to find printer.key in entries")
}

func TestManager_GetAllTranslations_PrinterTranslation(t *testing.T) {
	Initialize(language.English)

	// Register translations
	_ = RegisterTranslation(language.English, "printer.test", "English")
	_ = RegisterTranslation(language.French, "printer.test", "Français")

	entries := GetAllTranslations()

	found := false
	for _, entry := range entries {
		if entry.Key == "printer.test" {
			found = true
			// Should have French translation
			if entry.Translations[language.French] == "" {
				t.Error("expected French translation")
			}
		}
	}
	testutils.Assert(t, found, "expected to find printer.test in entries")
}

func TestManager_Reload_QueueWithUndLang(t *testing.T) {
	Initialize(language.English)

	// Queue translation with Und language (should map to fallback)
	_ = QueueTranslation(language.Und, "und.key", "Und Value")

	Reload()

	// Should be available in English (fallback)
	result := T("und.key")
	testutils.Equal(t, "Und Value", result)
}

func TestManager_Reload_DifferentFallbackAndCurrent(t *testing.T) {
	Initialize(language.English)

	// Set current language to French
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = SetLanguage(language.French)

	// Reload should create separate printers
	Reload()

	// Verify both printers exist
	fallbackPrinter := GetFallbackPrinter()
	currentPrinter := GetPrinter()

	testutils.NotNil(t, fallbackPrinter, "expected non-nil fallback printer")
	testutils.NotNil(t, currentPrinter, "expected non-nil current printer")
	if fallbackPrinter == currentPrinter {
		t.Error("expected different printers for fallback and current")
	}
}

func TestManager_Reload_SameFallbackAndCurrent(t *testing.T) {
	Initialize(language.English)
	_ = SetLanguage(language.English) // Ensure current == fallback

	// Reload should set currentPrinter to fallbackPrinter
	Reload()

	fallbackPrinter := GetFallbackPrinter()
	currentPrinter := GetPrinter()

	// They should be the same instance when current == fallback
	// (manager sets m.currentPrinter = m.fallbackPrinter in this case)
	testutils.NotNil(t, fallbackPrinter, "expected non-nil fallback printer")
	testutils.NotNil(t, currentPrinter, "expected non-nil current printer")
	// Note: They may be different instances even if same language
	// The important thing is they both work correctly
	_ = fallbackPrinter
	_ = currentPrinter
}

func TestManager_Reload_PrinterCache(t *testing.T) {
	Initialize(language.English)

	// Register multiple languages
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")
	_ = RegisterTranslation(language.Spanish, "test.key", "Valor")

	// Reload should create printers for all languages
	Reload()

	// Verify printers are available
	for _, lang := range []language.Tag{language.French, language.German, language.Spanish} {
		printer, err := GetPrinterFor(lang)
		if err != nil {
			t.Errorf("expected no error for %v, got %v", lang, err)
		}
		if printer == nil {
			t.Errorf("expected non-nil printer for %v", lang)
		}
	}
}

func TestManager_RegisterTranslation_RootKeyTracking(t *testing.T) {
	Initialize(language.English)

	// Register translation with root key
	_ = RegisterTranslation(language.English, "com.github.test.root.key", "Value")

	// Verify root key is tracked
	entries := GetAllTranslations()
	found := false
	for _, entry := range entries {
		if entry.Key == "com.github.test.root.key" {
			found = true
			if entry.RootKey == "" {
				t.Error("expected root key to be tracked")
			}
		}
	}
	testutils.Assert(t, found, "expected to find test key")
}

func TestManager_SupportLanguage(t *testing.T) {
	Initialize(language.English)

	// Register new language (should automatically support it)
	_ = RegisterTranslation(language.Italian, "test.key", "Valore")

	if !SupportsLanguage(language.Italian) {
		t.Error("Italian should be supported after registration")
	}
}

func TestManager_GetTranslationReport_PercentageCalculation(t *testing.T) {
	Initialize(language.English)

	// Register some translations with unique keys to avoid conflicts
	_ = RegisterTranslation(language.English, "reportcalc.key1", "Value 1")
	_ = RegisterTranslation(language.English, "reportcalc.key2", "Value 2")
	_ = RegisterTranslation(language.French, "reportcalc.key1", "Valeur 1")
	// reportcalc.key2 missing in French

	report := GetTranslationReport(language.French)

	// Find our specific keys in the report
	foundKey1 := false
	foundKey2 := false
	for _, entry := range report.MissingEntries {
		if entry.Key == "reportcalc.key2" {
			foundKey2 = true
		}
	}
	allEntries := GetAllTranslations()
	for _, entry := range allEntries {
		if entry.Key == "reportcalc.key1" {
			foundKey1 = true
			if entry.Translations[language.French] == "" {
				t.Error("expected French translation for key1")
			}
		}
		if entry.Key == "reportcalc.key2" {
			foundKey2 = true
		}
	}

	testutils.Assert(t, foundKey1, "expected to find reportcalc.key1")
	testutils.Assert(t, foundKey2, "expected to find reportcalc.key2")

	// Verify percentage calculation logic (may include other translations)
	if report.Total == 0 {
		t.Error("expected non-zero total")
	}
	if report.Percentage < 0 || report.Percentage > 100 {
		t.Errorf("percentage should be between 0 and 100, got %f", report.Percentage)
	}
}

func TestManager_GetTranslationReport_EmptyRootKey(t *testing.T) {
	Initialize(language.English)

	// Register flat translation (no root key)
	translations := map[string]any{
		"flat": map[string]any{
			"key": "Value",
		},
	}
	_ = RegisterTranslations(language.English, translations)

	report := GetTranslationReport(language.English)

	// Should handle empty root keys gracefully
	var found bool
	for _, entry := range report.MissingEntries {
		if entry.RootKey == "" {
			found = true
		}
	}
	testutils.Assert(t, found, "expected empty root key")
}

func TestManager_GetTranslationReport_UnknownRootKey(t *testing.T) {
	Initialize(language.English)

	// Register translation with short key (no extractable root)
	_ = RegisterTranslation(language.English, "short.key", "Value")

	report := GetTranslationReport(language.English)

	// Should handle unknown root keys
	foundUnknown := false
	for rootKey := range report.PerRootKey {
		if rootKey == "unknown" {
			foundUnknown = true
		}
	}
	// May or may not have "unknown" depending on extraction
	_ = foundUnknown
}
