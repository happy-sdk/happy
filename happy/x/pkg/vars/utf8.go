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

const (
	emptyStr = ""

	utf8RuneError = '\uFFFD' // the "error" Rune or "Unicode replacement character"
	utf8RuneSelf  = 0x80     // characters below RuneSelf are represented as themselves in a single byte.
	utf8UTFMax    = 4        // maximum number of bytes of a UTF-8 encoded Unicode character.
	utf8MaxRune   = '\U0010FFFF'

	// These names of these constants are chosen to give nice alignment in the
	// table below. The first nibble is an index into acceptRanges or F for
	// special one-byte cases. The second nibble is the Rune length or the
	// Status for the special one-byte case.
	xx = 0xF1 // invalid: size 1
	as = 0xF0 // ASCII: size 1
	s1 = 0x02 // accept 0, size 2
	s2 = 0x13 // accept 1, size 3
	s3 = 0x03 // accept 0, size 3
	s4 = 0x23 // accept 2, size 3
	s5 = 0x34 // accept 3, size 4
	s6 = 0x04 // accept 0, size 4
	s7 = 0x44 // accept 4, size 4

	// The default lowest and highest continuation byte.
	locb = 0b10000000
	hicb = 0b10111111

	maskx = 0b00111111
	mask2 = 0b00011111
	mask3 = 0b00001111
	mask4 = 0b00000111
)

// utf8AcceptRange gives the range of valid values
// for the second byte in a UTF-8 sequence.
type utf8AcceptRange struct {
	lo uint8 // lowest value for second byte.
	hi uint8 // highest value for second byte.
}

var (
	// utfFirst is information about the first byte in a UTF-8 sequence.
	utf8first = [256]uint8{
		//   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x00-0x0F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x10-0x1F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x20-0x2F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x30-0x3F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x40-0x4F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x50-0x5F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x60-0x6F
		as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, // 0x70-0x7F
		//   1   2   3   4   5   6   7   8   9   A   B   C   D   E   F
		xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0x80-0x8F
		xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0x90-0x9F
		xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xA0-0xAF
		xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xB0-0xBF
		xx, xx, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, // 0xC0-0xCF
		s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, // 0xD0-0xDF
		s2, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s4, s3, s3, // 0xE0-0xEF
		s5, s6, s6, s6, s7, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, // 0xF0-0xFF
	}

	// utf8AcceptRanges has size 16 to avoid bounds checks in the code that uses it.
	utf8AcceptRanges = [16]utf8AcceptRange{
		0: {locb, hicb},
		1: {0xA0, hicb},
		2: {locb, 0x9F},
		3: {0x90, hicb},
		4: {locb, 0x8F},
	}
)

// DecodeLastRuneInString is like DecodeLastRune but its input is a string. If
// s is empty it returns (RuneError, 0). Otherwise, if the encoding is invalid,
// it returns (RuneError, 1). Both are impossible results for correct,
// non-empty UTF-8.
//
// An encoding is invalid if it is incorrect UTF-8, encodes a rune that is
// out of range, or is not the shortest possible UTF-8 encoding for the
// value. No other validation is performed.
func utf8DecodeLastRuneInString(s string) (r rune, size int) {
	end := len(s)
	if end == 0 {
		return utf8RuneError, 0
	}
	start := end - 1
	r = rune(s[start])
	if r < utf8RuneSelf {
		return r, 1
	}
	// guard against O(n^2) behavior when traversing
	// backwards through strings with long sequences of
	// invalid UTF-8.
	lim := end - utf8UTFMax
	if lim < 0 {
		lim = 0
	}
	for start--; start >= lim; start-- {
		if utf8RuneStart(s[start]) {
			break
		}
	}
	if start < 0 {
		start = 0
	}
	r, size = utf8DecodeRuneInString(s[start:end])
	if start+size != end {
		return utf8RuneError, 1
	}
	return r, size
}

// utf8RuneStart reports whether the byte could be the first byte of an encoded,
// possibly invalid rune. Second and subsequent bytes always have the top two
// bits set to 10.
func utf8RuneStart(b byte) bool { return b&0xC0 != 0x80 }

// DecodeRuneInString is like DecodeRune but its input is a string. If s is
// empty it returns (RuneError, 0). Otherwise, if the encoding is invalid, it
// returns (RuneError, 1). Both are impossible results for correct, non-empty
// UTF-8.
//
// An encoding is invalid if it is incorrect UTF-8, encodes a rune that is
// out of range, or is not the shortest possible UTF-8 encoding for the
// value. No other validation is performed.
func utf8DecodeRuneInString(s string) (r rune, size int) {
	n := len(s)
	if n < 1 {
		return utf8RuneError, 0
	}
	s0 := s[0]
	x := utf8first[s0]
	if x >= as {
		// The following code simulates an additional check for x == xx and
		// handling the ASCII and invalid cases accordingly. This mask-and-or
		// approach prevents an additional branch.
		mask := rune(x) << 31 >> 31 // Create 0x0000 or 0xFFFF.
		return rune(s[0])&^mask | utf8RuneError&mask, 1
	}
	sz := int(x & 7)
	accept := utf8AcceptRanges[x>>4]
	if n < sz {
		return utf8RuneError, 1
	}
	s1 := s[1]
	if s1 < accept.lo || accept.hi < s1 {
		return utf8RuneError, 1
	}
	if sz <= 2 { // <= instead of == to help the compiler eliminate some bounds checks
		return rune(s0&mask2)<<6 | rune(s1&maskx), 2
	}
	s2 := s[2]
	if s2 < locb || hicb < s2 {
		return utf8RuneError, 1
	}
	if sz <= 3 {
		return rune(s0&mask3)<<12 | rune(s1&maskx)<<6 | rune(s2&maskx), 3
	}
	s3 := s[3]
	if s3 < locb || hicb < s3 {
		return utf8RuneError, 1
	}
	return rune(s0&mask4)<<18 | rune(s1&maskx)<<12 | rune(s2&maskx)<<6 | rune(s3&maskx), 4
}

// RuneCountInString is like RuneCount but its input is a string.
func utf8RuneCountInString(s string) (n int) {
	ns := len(s)
	for i := 0; i < ns; n++ {
		c := s[i]
		if c < utf8RuneSelf {
			// ASCII fast path
			i++
			continue
		}
		x := utf8first[c]
		if x == xx {
			i++ // invalid.
			continue
		}
		size := int(x & 7)
		if i+size > ns {
			i++ // Short or invalid.
			continue
		}
		accept := utf8AcceptRanges[x>>4]
		if c := s[i+1]; c < accept.lo || accept.hi < c {
			size = 1
		} else if size == 2 {
		} else if c := s[i+2]; c < locb || hicb < c {
			size = 1
		} else if size == 3 {
		} else if c := s[i+3]; c < locb || hicb < c {
			size = 1
		}
		i += size
	}
	return n
}

const (
	rune1Max = 1<<7 - 1
	rune2Max = 1<<11 - 1
	rune3Max = 1<<16 - 1
)

// Code points in the surrogate range are not valid for UTF-8.
const (
	surrogateMin = 0xD800
	surrogateMax = 0xDFFF
)

const (
	tx = 0b10000000
	t2 = 0b11000000
	t3 = 0b11100000
	t4 = 0b11110000
)

// EncodeRune writes into p (which must be large enough) the UTF-8 encoding of the rune.
// If the rune is out of range, it writes the encoding of RuneError.
// It returns the number of bytes written.
func utf8EncodeRune(p []byte, r rune) int {
	// Negative values are erroneous. Making it unsigned addresses the problem.
	switch i := uint32(r); {
	case i <= rune1Max:
		p[0] = byte(r)
		return 1
	case i <= rune2Max:
		_ = p[1] // eliminate bounds checks
		p[0] = t2 | byte(r>>6)
		p[1] = tx | byte(r)&maskx
		return 2
	case i > utf8MaxRune, surrogateMin <= i && i <= surrogateMax:
		r = utf8RuneError
		fallthrough
	case i <= rune3Max:
		_ = p[2] // eliminate bounds checks
		p[0] = t3 | byte(r>>12)
		p[1] = tx | byte(r>>6)&maskx
		p[2] = tx | byte(r)&maskx
		return 3
	default:
		_ = p[3] // eliminate bounds checks
		p[0] = t4 | byte(r>>18)
		p[1] = tx | byte(r>>12)&maskx
		p[2] = tx | byte(r>>6)&maskx
		p[3] = tx | byte(r)&maskx
		return 4
	}
}

// ValidRune reports whether r can be legally encoded as UTF-8.
// Code points that are out of range or a surrogate half are illegal.
func utf8ValidRune(r rune) bool {
	switch {
	case 0 <= r && r < surrogateMin:
		return true
	case surrogateMax < r && r <= utf8MaxRune:
		return true
	}
	return false
}
