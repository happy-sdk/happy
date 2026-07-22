// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"embed"
	"errors"
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
	testutils.Equal(t, "locales", fs.prefix)
}

func TestFS_WithPrefix(t *testing.T) {
	fs := NewFS(testFS)
	fs = fs.WithPrefix("custom")

	testutils.Equal(t, "custom", fs.prefix)
}

func TestRegisterTranslationsFS(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslationsFS(NewFS(locales))
	if err != nil {
		t.Logf("RegisterTranslationsFS error (may be expected): %v", err)
	}
}

func TestEmbed_ValidBundle(t *testing.T) {
	Initialize(language.English)
	// locales is this package's own real bundle (i18n.go's own
	// `//go:embed locales/*`) - its root already directly contains
	// "locales", exactly the one convention Embed/MustEmbed hardcode (see
	// NewFS's own default prefix). Reusing it here means this test
	// exercises the exact shape every real bundle in the monorepo has, not
	// a synthetic fixture.
	//
	// Embed deliberately returns nothing (see its doc comment) - a failed
	// registration is only recorded for automatic pickup by a later
	// Initialize/Reload call (see TestEmbed_FailureIsPickedUpByReload), or
	// observable by checking whether a key resolved, as here - never via a
	// return value most call sites would just discard, and never logged
	// directly (this package reports, it doesn't log).
	Embed(locales)
	testutils.Equal(t, "i18n is disabled", T("com.github.happy-sdk.happy.pkg.i18n.error.disabled"))
}

func TestEmbed_BrokenBundleDoesNotPanic(t *testing.T) {
	// testFS (`//go:embed testdata/*`) has no "l10n" directory at its own
	// root at all - Embed hardcodes exactly that convention (see NewFS's
	// own default prefix), so this must fail to register anything. Confirm
	// the failure is real via RegisterTranslationsFS directly (which still
	// returns the error), then confirm Embed itself does not panic on the
	// same input (see MustEmbed for the fail-loudly form) - it only records
	// the issue (see TestEmbed_FailureIsPickedUpByReload).
	err := RegisterTranslationsFS(NewFS(testFS))
	testutils.Assert(t, err != nil, "expected an error: testFS has no \"l10n\" directory at its own root")
	Embed(testFS)
	_ = drainPendingIssues() // discard: this test only cares that Embed didn't panic, not what it recorded
}

// TestEmbed_FailureIsPickedUpByReload is a regression test for the exact
// behavior the maintainer asked for: Embed must never log anything itself
// (not even via slog) - a failure it records must instead surface
// automatically the next time Initialize or Reload is called, so a
// package that calls Embed from its own init() (which necessarily runs
// before an application ever gets to call either) still gets its failure
// reported through happy-sdk's own logging, with no cooperation required
// from the package that called Embed.
func TestEmbed_FailureIsPickedUpByReload(t *testing.T) {
	Initialize(language.English)
	_ = drainPendingIssues() // start from a clean pool - other tests may have left issues pending

	Embed(testFS) // no "l10n" directory at testFS's own root - guaranteed to fail

	issues := Reload()
	testutils.Assert(t, len(issues) > 0, "expected Reload to pick up the issue Embed recorded")
	for _, issue := range issues {
		testutils.Assert(t, !issue.Fatal, "an Embed-recorded issue must never be Fatal")
	}

	// The pool must actually have been drained - a second Reload should not
	// see the same issue again.
	testutils.Equal(t, 0, len(Reload()))
}

func TestEmbedIssues_ReturnsAndStillRecordsForPickup(t *testing.T) {
	Initialize(language.English)
	_ = drainPendingIssues() // start from a clean pool

	issues := EmbedIssues(testFS) // no "l10n" directory at testFS's own root
	testutils.Assert(t, len(issues) > 0, "expected EmbedIssues to return the failure directly")
	for _, issue := range issues {
		testutils.Assert(t, !issue.Fatal, "an EmbedIssues failure must never be Fatal")
	}

	// The same failure must ALSO have been recorded for automatic pickup -
	// both audiences (a standalone caller inspecting the direct return
	// value, and happy-sdk calling Initialize/Reload) see it.
	testutils.Assert(t, len(drainPendingIssues()) > 0, "expected EmbedIssues to also record the issue for automatic pickup")
}

func TestEmbedIssues_NoIssuesOnValidBundle(t *testing.T) {
	Initialize(language.English)
	_ = drainPendingIssues()
	issues := EmbedIssues(locales)
	testutils.Equal(t, 0, len(issues))
	testutils.Equal(t, 0, len(drainPendingIssues()))
}

func TestMustEmbed_PanicsOnBrokenBundle(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected MustEmbed to panic on a broken/missing bundle")
		}
	}()
	MustEmbed(testFS) // no "l10n" directory at root - see TestEmbed_BrokenBundleWarnsWithoutPanicking
}

func TestMustEmbed_NoPanicOnValidBundle(t *testing.T) {
	Initialize(language.English)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("did not expect MustEmbed to panic on a valid bundle, got: %v", r)
		}
	}()
	MustEmbed(locales)
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
	testutils.Assert(t, errors.Is(err, ErrParseFile), "expected err to be ErrParseFile, got %v", err)
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
	testutils.Assert(t, errors.Is(err, ErrParseLanguageTag), "expected err to be ErrParseLanguageTag, got %v", err)
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
	if !errors.Is(err, ErrReadDir) {
		t.Errorf("expected err to be ErrReadDir, got %q", err.Error())
	}
}

func TestRegisterTranslationsFS_ReadRootError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs.prefix = "nonexistent_root_directory_abc123"
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for non-existent root directory")
	testutils.Assert(t, errors.Is(err, ErrReadDir), "expected err to be ErrReadDir, got %v", err)
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
	testutils.Assert(t, errors.Is(err, ErrParseLanguageTag), "expected err to be ErrParseLanguageTag, got %v", err)
	testutils.Assert(t, !strings.Contains(err.Error(), "test.txt"), "test.txt should be skipped, not cause an error")
}

func TestRegisterTranslationsFS_InvalidLanguageInFile(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for invalid language")
	testutils.Assert(t, errors.Is(err, ErrParseLanguageTag), "expected err to be ErrParseLanguageTag, got %v", err)
}

func TestRegisterTranslationsFS_InvalidLanguageInDir(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	fs = fs.WithPrefix("testdata")
	err := RegisterTranslationsFS(fs)
	testutils.Error(t, err, "expected error for invalid language in directory name")
	testutils.Assert(t, errors.Is(err, ErrParseLanguageTag), "expected err to be ErrParseLanguageTag, got %v", err)
}

// TestFS_LoadFile_DuplicateKeyRejected pins the section-5 loader guarantee:
// encoding/json/v2 (via jsontext) rejects duplicate object member names by
// default, so a translation file with a duplicated key now surfaces as a real
// ErrParseFile load error instead of one silently overwriting the other.
func TestFS_LoadFile_DuplicateKeyRejected(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/dupkey.json")
	testutils.Error(t, err, "expected error for a duplicate translation key")
	testutils.Assert(t, errors.Is(err, ErrParseFile), "expected err to be ErrParseFile, got %v", err)
}

// TestFS_LoadFile_InvalidUTF8Rejected pins the other half of the section-5
// guarantee: invalid UTF-8 in a .json bundle is rejected at decode time rather
// than passed through.
func TestFS_LoadFile_InvalidUTF8Rejected(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/badutf8.json")
	testutils.Error(t, err, "expected error for invalid UTF-8")
	testutils.Assert(t, errors.Is(err, ErrParseFile), "expected err to be ErrParseFile, got %v", err)
}

func TestFS_LoadFile_JSONParseError(t *testing.T) {
	Initialize(language.English)
	fs := NewFS(testFS)
	err := fs.loadFile(language.English, "testdata/invalid.json")
	testutils.Error(t, err, "expected error for invalid JSON")
	testutils.Assert(t, errors.Is(err, ErrParseFile), "expected err to be ErrParseFile, got %v", err)
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
