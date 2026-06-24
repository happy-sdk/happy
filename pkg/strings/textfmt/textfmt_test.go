// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package textfmt

import "testing"

func TestRemoveNonPrintableChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text unchanged", "hello world", "hello world"},
		{"tabs stripped", "hello\tworld", "helloworld"},
		{"newlines stripped", "hello\nworld", "helloworld"},
		{"control chars stripped", "hello\x00\x01world", "helloworld"},
		{"unicode letters preserved", "héllo wörld", "héllo wörld"},
		{"empty string", "", ""},
		{"only control chars", "\t\n\r", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := RemoveNonPrintableChars(test.input)
			if got != test.want {
				t.Errorf("RemoveNonPrintableChars(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
