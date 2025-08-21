// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package textfmt offers a collection of utils for plain text formatting.
package textfmt

import (
	"strings"
	"unicode"
)

// RemoveNonPrintableChars removes non-printable characters
// Uses Go's unicode package to determine if a character is printable
func RemoveNonPrintableChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}
