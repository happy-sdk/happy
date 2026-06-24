// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package textfmt offers a collection of utils for plain text formatting.
package textfmt

import (
	"strings"
	"unicode"
)

// RemoveNonPrintableChars removes non-printable characters using Go's
// unicode.IsPrint to determine printability. This includes control
// characters such as tabs and newlines, which are stripped (not converted
// to spaces) -- unlike wordwrap.go's word-wrapping logic, which treats tab
// as meaningful whitespace for column alignment; the two functions serve
// different purposes and intentionally disagree on tab handling.
func RemoveNonPrintableChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}
