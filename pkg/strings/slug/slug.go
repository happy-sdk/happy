// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package slug generates URL-safe, hyphen-separated slugs from arbitrary
// input strings.
package slug

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// whitespaceRe matches any run of Unicode whitespace (spaces, tabs,
	// newlines, etc.), which is folded to a single separator.
	whitespaceRe = regexp.MustCompile(`[\s]+`)
	// separatorRunRe collapses any run of hyphens/underscores -- including
	// mixed runs like "-_" -- into a single hyphen.
	separatorRunRe = regexp.MustCompile(`[-_]+`)
	// invalidCharsRe strips anything that isn't a lowercase ASCII letter,
	// digit, or hyphen.
	invalidCharsRe = regexp.MustCompile(`[^a-z0-9-]+`)
)

// foldDiacritics maps common Latin letters with diacritical marks (from the
// Latin-1 Supplement and Latin Extended-A Unicode blocks) to their plain
// ASCII equivalent, so e.g. "héllo" becomes "hello" instead of being
// silently dropped by the ASCII-only filter applied afterwards. Scripts
// with no Latin transliteration (e.g. CJK, Cyrillic, Arabic) are not
// covered, and their characters are dropped -- consistent with most
// lightweight slug generators, which don't attempt full romanization.
var foldDiacritics = map[rune]string{
	'À': "a", 'Á': "a", 'Â': "a", 'Ã': "a", 'Ä': "a", 'Å': "a", 'Ā': "a", 'Ă': "a", 'Ą': "a",
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'Æ': "ae", 'æ': "ae",
	'Ç': "c", 'Ć': "c", 'Č': "c", 'ç': "c", 'ć': "c", 'č': "c",
	'Ð': "d", 'Ď': "d", 'Đ': "d", 'ð': "d", 'ď': "d", 'đ': "d",
	'È': "e", 'É': "e", 'Ê': "e", 'Ë': "e", 'Ē': "e", 'Ė': "e", 'Ę': "e", 'Ě': "e",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'Ì': "i", 'Í': "i", 'Î': "i", 'Ï': "i", 'Ī': "i", 'Į': "i",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i",
	'Ł': "l", 'Ĺ': "l", 'ł': "l", 'ĺ': "l",
	'Ñ': "n", 'Ń': "n", 'Ň': "n", 'ñ': "n", 'ń': "n", 'ň': "n",
	'Ò': "o", 'Ó': "o", 'Ô': "o", 'Õ': "o", 'Ö': "o", 'Ø': "o", 'Ō': "o",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o",
	'Œ': "oe", 'œ': "oe",
	'Ŕ': "r", 'Ř': "r", 'ŕ': "r", 'ř': "r",
	'Š': "s", 'Ś': "s", 'š': "s", 'ś': "s",
	'ß': "ss",
	'Ť': "t", 'Þ': "t", 'ť': "t", 'þ': "t",
	'Ù': "u", 'Ú': "u", 'Û': "u", 'Ü': "u", 'Ū': "u", 'Ů': "u",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u",
	'Ý': "y", 'Ÿ': "y", 'ý': "y", 'ÿ': "y",
	'Ž': "z", 'Ź': "z", 'Ż': "z", 'ž': "z", 'ź': "z", 'ż': "z",
}

// Create generates a URL-safe, hyphen-separated slug from the provided
// string: it lowercases the input, transliterates common Latin diacritics
// (e.g. "é" -> "e") to ASCII, folds whitespace and any run of hyphens or
// underscores into a single hyphen, strips any remaining character that
// isn't a lowercase ASCII letter, digit, or hyphen, and trims leading or
// trailing hyphens.
//
// Characters from scripts with no Latin transliteration (e.g. CJK,
// Cyrillic, Arabic) are dropped; if input consists entirely of such
// characters, Create returns "".
func Create(input string) string {
	var b strings.Builder
	for _, r := range input {
		if folded, ok := foldDiacritics[r]; ok {
			b.WriteString(folded)
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}

	str := whitespaceRe.ReplaceAllString(b.String(), "-")
	// Collapse separator runs (including mixed "-_" runs) to a single
	// hyphen before stripping invalid characters, since the strip step
	// doesn't treat "_" as valid on its own.
	str = separatorRunRe.ReplaceAllString(str, "-")
	str = invalidCharsRe.ReplaceAllString(str, "")
	// Stripping invalid characters can join two separators that were
	// previously apart only by now-removed characters (e.g. "a-!-b"
	// becomes "a--b"), so collapse separator runs once more.
	str = separatorRunRe.ReplaceAllString(str, "-")
	return strings.Trim(str, "-")
}

// IsValid reports whether slug is a valid slug as produced by Create: a
// non-empty, lowercase ASCII string of letters, digits, and single
// (non-repeated) hyphens, with no leading or trailing hyphen.
func IsValid(slug string) bool {
	if slug == "" || strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") || strings.Contains(slug, "--") {
		return false
	}
	return !invalidCharsRe.MatchString(slug)
}
