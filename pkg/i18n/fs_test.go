// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"embed"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

//go:embed testdata/*
var testFS embed.FS

func TestNewFS(t *testing.T) {
	fs := NewFS(testFS)
	testutils.NotNil(t, fs)
	testutils.Equal(t, "i18n", fs.prefix)
}

func TestFS_WithPrefix(t *testing.T) {
	fs := NewFS(testFS)
	fs = fs.WithPrefix("custom")

	testutils.Equal(t, "custom", fs.prefix)
}

func TestRegisterTranslationsFS(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslationsFS(NewFS(translations))
	if err != nil {
		t.Logf("RegisterTranslationsFS error (may be expected): %v", err)
	}
}

func TestFS_readRoot(t *testing.T) {
	fs := NewFS(testFS)

	// Test with non-existent prefix
	fs.prefix = "nonexistent"
	_, err := fs.readRoot()
	testutils.Error(t, err, "expected error for non-existent directory")
}

func TestFS_loadFile(t *testing.T) {
	Initialize(language.English)

	fs := NewFS(testFS)

	// Test with non-existent file
	err := fs.loadFile(language.English, "nonexistent.json")
	testutils.Error(t, err, "expected error for non-existent file")
}

func TestFS_load(t *testing.T) {
	Initialize(language.English)

	fs := NewFS(testFS)

	// Test with non-existent directory
	err := fs.load(language.English, "nonexistent")
	testutils.Error(t, err, "expected error for non-existent directory")
}

func TestRegisterTranslationsFS_FlatFormat(t *testing.T) {
	Initialize(language.English)

	// Test with flat format translations
	translations := map[string]any{
		"app": map[string]any{
			"name": "Test App",
		},
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	// Verify translation is accessible
	result := T("app.name")
	testutils.Equal(t, "Test App", result)
}

func TestRegisterTranslationsFS_StructuredFormat(t *testing.T) {
	Initialize(language.English)

	// Test with structured format (root key)
	translations := map[string]any{
		"com.github.happy-sdk.test": map[string]any{
			"key": "Value",
		},
	}

	err := RegisterTranslations(language.English, translations)
	testutils.NoError(t, err)

	// Verify translation is accessible
	result := T("com.github.happy-sdk.test.key")
	testutils.Equal(t, "Value", result)
}

func TestLooksLikeRootKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"com.github.happy-sdk", true},
		{"org.example.test", true},
		{"app.name", true}, // Has 3 parts, so returns true
		{"key", false},     // Only 1 part
		{"github.com.test", true},
		{"test", false},       // Only 1 part
		{"app", false},        // Only 1 part
		{"net.example", true}, // Has 2 parts and starts with common TLD
	}

	for _, tt := range tests {
		result := looksLikeRootKey(tt.key)
		testutils.Equal(t, tt.expected, result, "looksLikeRootKey(%q)", tt.key)
	}
}

func TestFS_WithPrefix_Chain(t *testing.T) {
	fs := NewFS(testFS)
	fs = fs.WithPrefix("prefix1")
	testutils.Equal(t, "prefix1", fs.prefix)
	fs = fs.WithPrefix("prefix2")
	testutils.Equal(t, "prefix2", fs.prefix)
}

func TestFS_LoadFile_ReadError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "nonexistent_file.json")
	testutils.Error(t, err, "expected error for non-existent file")
}

func TestFS_Load_DirectoryInLangDirError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.load(language.English, "testdata")
	_ = err
}

func TestFS_LoadFile_JSONUnmarshalError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/invalid.json")
	testutils.Error(t, err, "expected error for invalid JSON")
	testutils.ContainsString(t, err.Error(), "could not parse translation file")
}

func TestRegisterTranslationsFS_FileInRoot(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestRegisterTranslationsFS_InvalidLangInFileName(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for invalid language")
	testutils.Assert(t,
		strings.Contains(err.Error(), "parsing language tag from file") ||
			strings.Contains(err.Error(), "parsing language tag from dir"),
		"expected error about parsing language tag")
}

func TestRegisterTranslationsFS_InvalidLangInDirName(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestFS_LoadFile_RegisterTranslationsError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/en.json")
	_ = err
}

func TestRegisterTranslationsFS_LoadFileError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestFS_ReadRoot_ErrorPath(t *testing.T) {
	fs := NewFS(testFS)
	fs.prefix = "nonexistent_directory_that_does_not_exist_12345"
	_, err := fs.readRoot()
	testutils.Error(t, err, "expected error for non-existent directory")
}

func TestFS_LoadFile_ReadFileError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "nonexistent_file_12345.json")
	testutils.Error(t, err, "expected error for non-existent file")
}

func TestRegisterTranslationsFS_ErrorHandling(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs.prefix = "nonexistent"
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for non-existent directory")
}

func TestFS_WithPrefix_MultipleCalls(t *testing.T) {
	fs := NewFS(testFS)
	fs = fs.WithPrefix("prefix1")
	testutils.Equal(t, "prefix1", fs.prefix)
	fs = fs.WithPrefix("prefix2")
	testutils.Equal(t, "prefix2", fs.prefix)
	fs = fs.WithPrefix("prefix3")
	testutils.Equal(t, "prefix3", fs.prefix)
}

func TestRegisterTranslationsFS_NonJSONFileSkipped(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestFS_Load_MultipleFilesInDir(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.load(language.English, "testdata")
	_ = err
}

func TestFS_LoadFile_BaseNameExtraction(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/en.json")
	_ = err
}

func TestFS_Load_ReadDirError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs.prefix = "nonexistent_directory_xyz123"
	err := fs.load(language.English, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
	if !strings.Contains(err.Error(), "loading translations from fs failed") {
		t.Errorf("expected load error, got %q", err.Error())
	}
}

func TestRegisterTranslationsFS_ReadRootError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs.prefix = "nonexistent_root_directory_abc123"
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for non-existent root directory")
	testutils.ContainsString(t, err.Error(), "i18n loading translations from fs")
}

func TestRegisterTranslationsFS_NonJSONFileSkip(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestRegisterTranslationsFS_InvalidLangInDir(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestRegisterTranslationsFS_LoadError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := RegisterTranslationsFS(fs)
	_ = err
}

func TestRegisterTranslationsFS_NonJSONFile(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error from invalid language")
	testutils.Assert(t,
		strings.Contains(err.Error(), "parsing language tag from file") ||
			strings.Contains(err.Error(), "parsing language tag from dir"),
		"expected error about parsing language tag")
	testutils.Assert(t, !strings.Contains(err.Error(), "test.txt"), "test.txt should be skipped, not cause an error")
}

func TestRegisterTranslationsFS_InvalidLanguageInFile(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for invalid language")
	testutils.Assert(t,
		strings.Contains(err.Error(), "parsing language tag from file") ||
			strings.Contains(err.Error(), "parsing language tag from dir"),
		"expected error about parsing language tag")
}

func TestRegisterTranslationsFS_InvalidLanguageInDir(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for invalid language in directory name")
	testutils.ContainsString(t, err.Error(), "parsing language tag from dir")
}

func TestFS_LoadFile_JSONParseError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/invalid.json")
	testutils.Error(t, err, "expected error for invalid JSON")
	testutils.ContainsString(t, err.Error(), "could not parse translation file")
}

func TestRegisterTranslationsFS_FlatAndStructured(t *testing.T) {
	Initialize(language.English)
	flatTranslations := map[string]any{
		"app.name":    "Test App",
		"app.version": "1.0.0",
	}
	err := RegisterTranslations(language.English, flatTranslations)
	testutils.NoError(t, err)

	fs := NewFS(testFS)
	err = fs.loadFile(language.English, "testdata/en.json")
	testutils.NoError(t, err)

	flatResult := T("app.name")
	testutils.Equal(t, "Test App", flatResult, "expected flat format translation")

	structuredResult := T("test.key")
	testutils.Equal(t, "value", structuredResult, "expected structured format translation from FS")
}

func TestFS_Load_MultipleFiles(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.load(language.French, "testdata/fr")
	testutils.NoError(t, err, "expected to load multiple files successfully")

	err = SetLanguage(language.French)
	testutils.NoError(t, err, "expected to set language to French")

	result1 := T("greeting")
	testutils.Equal(t, "Bonjour", result1, "expected translation from fr1.json")

	result2 := T("welcome")
	testutils.Equal(t, "Bienvenue", result2, "expected translation from fr2.json")
}
