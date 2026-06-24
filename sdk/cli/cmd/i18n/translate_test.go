// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"reflect"
	"testing"
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
