// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package slug

import "testing"

func TestCreate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "Hello World", "hello-world"},
		{"already slugified", "hello-world", "hello-world"},
		{"leading/trailing spaces", "  hello world  ", "hello-world"},
		{"multiple spaces", "hello    world", "hello-world"},
		{"multiple hyphens", "hello---world", "hello-world"},
		{"underscore separator", "hello_world", "hello-world"},
		{"multiple underscores", "hello___world", "hello-world"},
		{"mixed hyphen/underscore run", "hello-_world", "hello-world"},
		{"mixed underscore/hyphen run", "hello_-world", "hello-world"},
		{"separator joined by stripped char", "hello-!-world", "hello-world"},
		{"tabs and newlines", "hello\tworld\ncode", "hello-world-code"},
		{"punctuation stripped", "hello, world!", "hello-world"},
		{"leading/trailing separators trimmed", "-hello-world-", "hello-world"},
		{"leading/trailing underscores trimmed", "_hello_world_", "hello-world"},
		{"digits preserved", "page 2 of 10", "page-2-of-10"},
		{"empty input", "", ""},
		{"only invalid chars", "!!!", ""},
		{"latin diacritics", "héllo wörld", "hello-world"},
		{"latin diacritics uppercase", "HÉLLO WÖRLD", "hello-world"},
		{"french", "Crème brûlée", "creme-brulee"},
		{"german eszett", "straße", "strasse"},
		{"non-latin script dropped entirely", "日本語のテスト", ""},
		{"mixed latin and non-latin", "hello 日本語 world", "hello-world"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Create(test.input)
			if got != test.want {
				t.Errorf("Create(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

// TestCreateOutputAlwaysValid is a consistency check between Create and
// IsValid: addon registration (sdk/addon) calls slug.Create to derive a
// slug and then validates it with slug.IsValid, so any non-empty slug
// Create produces must satisfy IsValid, or registration breaks for inputs
// that happen to trip up one function but not the other.
func TestCreateOutputAlwaysValid(t *testing.T) {
	inputs := []string{
		"Hello World", "hello-world", "hello_world", "hello-_world",
		"  spaced  ", "Tab\tSeparated", "MiXeD CaSe", "héllo wörld",
		"123", "a", "-", "_", "", "!!!", "日本語",
	}
	for _, in := range inputs {
		got := Create(in)
		if got == "" {
			continue
		}
		if !IsValid(got) {
			t.Errorf("Create(%q) = %q, which IsValid rejects", in, got)
		}
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		slug string
		want bool
	}{
		{"hello-world", true},
		{"hello", true},
		{"hello123", true},
		{"", false},
		{"-hello", false},
		{"hello-", false},
		{"hello--world", false},
		{"hello_world", false},
		{"Hello-World", false},
		{"hello world", false},
		{"hello.world", false},
		{"日本語", false},
	}
	for _, test := range tests {
		t.Run(test.slug, func(t *testing.T) {
			if got := IsValid(test.slug); got != test.want {
				t.Errorf("IsValid(%q) = %v, want %v", test.slug, got, test.want)
			}
		})
	}
}
