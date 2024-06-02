// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package slug

import (
	"regexp"
	"strings"
)

// Create generates a URL-safe slug from the provided string
func Create(input string) string {
	str := strings.ToLower(input)
	spacelessStr := strings.ReplaceAll(str, " ", "-")
	reg := regexp.MustCompile(`[^a-z0-9-_]+`)
	cleanedStr := reg.ReplaceAllString(spacelessStr, "")

	// Replace multiples
	doubleHyphenRe := regexp.MustCompile(`-+`)
	slug := doubleHyphenRe.ReplaceAllString(cleanedStr, "-")
	doubleUnderscoreRe := regexp.MustCompile(`_+`)
	slug = doubleUnderscoreRe.ReplaceAllString(slug, "_")
	slug = strings.Trim(slug, "-_")
	return slug
}

// IsValid checks if the provided slug is valid based on the rules
func IsValid(slug string) bool {
	// Regular expression to check for invalid characters
	invalidRe := regexp.MustCompile(`[^a-z0-9-_]`)
	if invalidRe.MatchString(slug) {
		return false
	}

	if strings.Contains(slug, "--") || strings.Contains(slug, "__") {
		return false
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") ||
		strings.HasPrefix(slug, "_") || strings.HasSuffix(slug, "_") {
		return false
	}
	return true
}
