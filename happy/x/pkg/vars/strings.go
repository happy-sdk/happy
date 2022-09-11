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

package vars

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

type stringer interface {
	String() string
}

// TrimSpace returns a slice of the string s, with all leading
// and trailing white space removed, as defined by Unicode.
func stringsTrimSpace(s string) string {
	// Fast path for ASCII: look for the first ASCII non-space byte
	start := 0
	for ; start < len(s); start++ {
		c := s[start]
		if c >= utf8RuneSelf {
			// If we run into a non-ASCII byte, fall back to the
			// slower unicode-aware method on the remaining bytes
			return stringsTrimFunc(s[start:], unicodeIsSpace)
		}
		if asciiSpace[c] == 0 {
			break
		}
	}

	// Now look for the first ASCII non-space byte from the end
	stop := len(s)
	for ; stop > start; stop-- {
		c := s[stop-1]
		if c >= utf8RuneSelf {
			// start has been already trimmed above, should trim end only
			return stringsTrimRightFunc(s[start:stop], unicodeIsSpace)
		}
		if asciiSpace[c] == 0 {
			break
		}
	}

	// At this point s[start:stop] starts and ends with an ASCII
	// non-space bytes, so we're done. Non-ASCII cases have already
	// been handled above.
	return s[start:stop]
}

// TrimFunc returns a slice of the string s with all leading
// and trailing Unicode code points c satisfying f(c) removed.
func stringsTrimFunc(s string, f func(rune) bool) string {
	return stringsTrimRightFunc(stringsTrimLeftFunc(s, f), f)
}

// TrimLeftFunc returns a slice of the string s with all leading
// Unicode code points c satisfying f(c) removed.
func stringsTrimLeftFunc(s string, f func(rune) bool) string {
	i := stringsIndexFunc(s, f, false)
	if i == -1 {
		return ""
	}
	return s[i:]
}

// TrimRightFunc returns a slice of the string s with all trailing
// Unicode code points c satisfying f(c) removed.
func stringsTrimRightFunc(s string, f func(rune) bool) string {
	i := stringsLastIndexFunc(s, f, false)
	if i >= 0 && s[i] >= utf8RuneSelf {
		_, wid := utf8DecodeRuneInString(s[i:])
		i += wid
	} else {
		i++
	}
	return s[0:i]
}

// lastIndexFunc is the same as LastIndexFunc except that if
// truth==false, the sense of the predicate function is
// inverted.
func stringsLastIndexFunc(s string, f func(rune) bool, truth bool) int {
	for i := len(s); i > 0; {
		r, size := utf8DecodeLastRuneInString(s[0:i])
		i -= size
		if f(r) == truth {
			return i
		}
	}
	return -1
}

// indexFunc is the same as IndexFunc except that if
// truth==false, the sense of the predicate function is
// inverted.
func stringsIndexFunc(s string, f func(rune) bool, truth bool) int {
	for i, r := range s {
		if f(r) == truth {
			return i
		}
	}
	return -1
}

// Fields splits the string s around each instance of one or more consecutive white space
// characters, as defined by unicode.IsSpace, returning a slice of substrings of s or an
// empty slice if s contains only white space.
func stringsFields(s string) []string {
	// First count the fields.
	// This is an exact count if s is ASCII, otherwise it is an approximation.
	n := 0
	wasSpace := 1
	// setBits is used to track which bits are set in the bytes of s.
	setBits := uint8(0)
	for i := 0; i < len(s); i++ {
		r := s[i]
		setBits |= r
		isSpace := int(asciiSpace[r])
		n += wasSpace & ^isSpace
		wasSpace = isSpace
	}

	if setBits >= utf8RuneSelf {
		// Some runes in the input string are not ASCII.
		return stringsFieldsFunc(s, unicodeIsSpace)
	}
	// ASCII fast path
	a := make([]string, n)
	na := 0
	fieldStart := 0
	i := 0
	// Skip spaces in the front of the input.
	for i < len(s) && asciiSpace[s[i]] != 0 {
		i++
	}
	fieldStart = i
	for i < len(s) {
		if asciiSpace[s[i]] == 0 {
			i++
			continue
		}
		a[na] = s[fieldStart:i]
		na++
		i++
		// Skip spaces in between fields.
		for i < len(s) && asciiSpace[s[i]] != 0 {
			i++
		}
		fieldStart = i
	}
	if fieldStart < len(s) { // Last field might end at EOF.
		a[na] = s[fieldStart:]
	}
	return a
}

// FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c)
// and returns an array of slices of s. If all code points in s satisfy f(c) or the
// string is empty, an empty slice is returned.
//
// FieldsFunc makes no guarantees about the order in which it calls f(c)
// and assumes that f always returns the same value for a given c.
func stringsFieldsFunc(s string, f func(rune) bool) []string {
	// A span is used to record a slice of s of the form s[start:end].
	// The start index is inclusive and the end index is exclusive.
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	// Doing this in a separate pass (rather than slicing the string s
	// and collecting the result substrings right away) is significantly
	// more efficient, possibly due to cache effects.
	start := -1 // valid span start if >= 0
	for end, rune := range s {
		if f(rune) {
			if start >= 0 {
				spans = append(spans, span{start, end})
				// Set start to a negative value.
				// Note: using -1 here consistently and reproducibly
				// slows down this code by a several percent on amd64.
				start = ^start
			}
		} else {
			if start < 0 {
				start = end
			}
		}
	}

	// Last field might end at EOF.
	if start >= 0 {
		spans = append(spans, span{start, len(s)})
	}

	// Create strings from recorded field indices.
	a := make([]string, len(spans))
	for i, span := range spans {
		a[i] = s[span.start:span.end]
	}

	return a
}

func stringsFastCut(s string, sep rune) (before, after string, found bool) {
	ix := -1
	for i, c := range s {
		if c == sep {
			ix = i
			break
		}
	}
	if ix > -1 {
		return s[:ix], s[ix+1:], true
	}
	return s, "", false
}

// strings
// Split slices s into all substrings separated by sep and returns a slice of
// the substrings between those separators.
//
// If s does not contain sep and sep is not empty, Split returns a
// slice of length 1 whose only element is s.
//
// If sep is empty, Split splits after each UTF-8 sequence. If both s
// and sep are empty, Split returns an empty slice.
//
// It is equivalent to SplitN with a count of -1.
//
// To split around the first instance of a separator, see Cut.
func stringsSplit(s string, sep rune) (ss []string) {
	for b, a, f := stringsFastCut(s, sep); f; b, a, f = stringsFastCut(a, sep) {
		ss = append(ss, b)
	}
	if len(ss) == 0 {
		return []string{s}
	}
	return
}
