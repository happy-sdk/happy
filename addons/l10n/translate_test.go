// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/text/language"
)

func TestExtractRootKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"a.b.c", "a.b.c"}, // too short (< 5 parts), returned as-is
		{"com.github.happy-sdk.happy.cli", "com.github.happy-sdk.happy.cli"},
		{"com.github.happy-sdk.happy.sdk.cli.flags.version", "com.github.happy-sdk.happy.sdk.cli"},
		{"com.github.happy-sdk.happy.pkg.vars.varflag.foo", "com.github.happy-sdk.happy.pkg.vars"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := extractRootKey(tt.key)
			if got != tt.want {
				t.Errorf("extractRootKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	t.Run("creates nested structure", func(t *testing.T) {
		m := map[string]any{}
		setNestedValue(m, "cmd.help.description", "value")

		want := map[string]any{
			"cmd": map[string]any{
				"help": map[string]any{
					"description": "value",
				},
			},
		}
		if !reflect.DeepEqual(m, want) {
			t.Errorf("setNestedValue result = %#v, want %#v", m, want)
		}
	})

	t.Run("reuses existing nested map", func(t *testing.T) {
		m := map[string]any{
			"cmd": map[string]any{
				"existing": "other",
			},
		}
		setNestedValue(m, "cmd.help", "value")

		cmd, ok := m["cmd"].(map[string]any)
		if !ok {
			t.Fatal("expected cmd to remain a map")
		}
		if cmd["existing"] != "other" {
			t.Error("expected existing sibling key to be preserved")
		}
		if cmd["help"] != "value" {
			t.Error("expected help to be set to value")
		}
	})

	t.Run("single-level key", func(t *testing.T) {
		m := map[string]any{}
		setNestedValue(m, "key", "value")
		if m["key"] != "value" {
			t.Errorf("expected m[key] = value, got %v", m["key"])
		}
	})
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	return m
}

func TestWriteAppTranslationPerLang(t *testing.T) {
	dir := t.TempDir()
	l10nDir := filepath.Join(dir, "l10n")
	cnf := defaultL10nFileConfig()

	path, err := writeAppTranslation(l10nDir, cnf, language.Estonian, "help.description", "Abi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(l10nDir, "et.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	m := readJSONFile(t, path)
	help, ok := m["help"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested help map, got %#v", m)
	}
	if help["description"] != "Abi" {
		t.Errorf("help.description = %v, want %q", help["description"], "Abi")
	}
}

func TestWriteAppTranslationPerLangPreservesSiblingKeys(t *testing.T) {
	dir := t.TempDir()
	l10nDir := filepath.Join(dir, "l10n")
	cnf := defaultL10nFileConfig()

	if _, err := writeAppTranslation(l10nDir, cnf, language.German, "help.description", "Hilfe"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path, err := writeAppTranslation(l10nDir, cnf, language.German, "help.usage", "Verwendung")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := readJSONFile(t, path)
	help := m["help"].(map[string]any)
	if help["description"] != "Hilfe" {
		t.Errorf("expected earlier sibling key help.description to survive, got %v", help["description"])
	}
	if help["usage"] != "Verwendung" {
		t.Errorf("help.usage = %v, want %q", help["usage"], "Verwendung")
	}
}

func TestWriteAppTranslationUnifiedPreservesOtherLanguages(t *testing.T) {
	dir := t.TempDir()
	l10nDir := filepath.Join(dir, "l10n")
	cnf := l10nFileConfig{Layout: string(layoutUnified)}

	if _, err := writeAppTranslation(l10nDir, cnf, language.German, "help.description", "Hilfe"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path, err := writeAppTranslation(l10nDir, cnf, language.Estonian, "help.description", "Abi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(l10nDir, defaultUnifiedFilename); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	m := readJSONFile(t, path)

	de, ok := m["de"].(map[string]any)
	if !ok {
		t.Fatalf("expected the earlier-saved German entry to still be present, got %#v", m)
	}
	deHelp := de["help"].(map[string]any)
	if deHelp["description"] != "Hilfe" {
		t.Errorf("expected German translation saved first to survive the Estonian save, got %v", deHelp["description"])
	}

	et := m["et"].(map[string]any)
	etHelp := et["help"].(map[string]any)
	if etHelp["description"] != "Abi" {
		t.Errorf("et.help.description = %v, want %q", etHelp["description"], "Abi")
	}
}

func TestWriteAppTranslationUnifiedCustomFilename(t *testing.T) {
	dir := t.TempDir()
	l10nDir := filepath.Join(dir, "l10n")
	cnf := l10nFileConfig{Layout: string(layoutUnified), Filename: "strings.json"}

	path, err := writeAppTranslation(l10nDir, cnf, language.French, "greeting", "Bonjour")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(l10nDir, "strings.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestWriteDependencyTranslationPreservesOtherEntries(t *testing.T) {
	dir := t.TempDir()
	l10nDir := filepath.Join(dir, "l10n")

	// Root keys deliberately contain no dots here: setNestedValue (exercised
	// separately in TestSetNestedValue) splits the whole "rootKey.lang.key"
	// path on ".", so a dotted root key like a real reverse-DNS module
	// identifier nests one map level per segment rather than staying a
	// single top-level key. That's existing, unchanged behavior - this test
	// is only about writeDependencyTranslation's read-modify-write, so it
	// sidesteps that by using single-segment root keys.
	rootA := "vendorone"
	rootB := "vendortwo"

	if _, err := writeDependencyTranslation(l10nDir, rootA, language.German, "flags.version", "Version"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path, err := writeDependencyTranslation(l10nDir, rootB, language.German, "greeting", "Hallo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(l10nDir, "dependencies.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	m := readJSONFile(t, path)
	a := m[rootA].(map[string]any)["de"].(map[string]any)
	if a["flags"].(map[string]any)["version"] != "Version" {
		t.Errorf("expected earlier dependency entry for %s to survive", rootA)
	}
	b := m[rootB].(map[string]any)["de"].(map[string]any)
	if b["greeting"] != "Hallo" {
		t.Errorf("greeting = %v, want %q", b["greeting"], "Hallo")
	}
}
