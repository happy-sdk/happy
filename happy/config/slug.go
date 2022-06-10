// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"
)

// Slug function returns slugifies string "s".
func CreateSlug(s string) string {
	for _, r := range rExps {
		s = r.re.ReplaceAllString(s, r.ch)
	}

	s = strings.ToLower(s)
	s = spacereg.ReplaceAllString(s, "-")
	s = noncharreg.ReplaceAllString(s, "")
	s = minusrepeatreg.ReplaceAllString(s, "-")

	return s
}

// ValidSlug returns true if s is string which is valid slug.
func ValidSlug(s string) bool {
	if len(s) == 1 {
		return unicode.IsLetter(rune(s[0]))
	}
	re := regexp.MustCompile(SlugRe)
	return re.MatchString(s)
}

// CreateCamelCaseSlug returns a camel case representation of the string all
// non alpha numeric characters removed. Uppercase characters are mapped
// first alnum in string and after each non alnum character is removed.
func CreateCamelCaseSlug(s string) string {
	var b bytes.Buffer
	tu := true
	for _, c := range s {
		isAlnum := unicode.Is(alnum, c)
		isSpace := unicode.IsSpace(c)
		isLower := unicode.IsLower(c)
		if isSpace || !isAlnum {
			tu = true
			continue
		}
		if tu {
			if isLower {
				b.WriteRune(unicode.ToUpper(c))
			} else {
				b.WriteRune(c)
			}
			tu = false

			continue
		} else {
			if !isLower {
				c = unicode.ToLower(c)
			}
			b.WriteRune(c)
		}
	}
	return b.String()
}
