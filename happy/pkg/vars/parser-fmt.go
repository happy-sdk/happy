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

import (
	"strconv"
)

func (f *parserFmt) init(buf *parserBuffer) {
	f.buf = buf
	f.clearflags()
}

func (f *parserFmt) clearflags() {
	f.parserFmtFlags = parserFmtFlags{plus: false}
}

// padString appends s to f.buf, padded on left (!f.minus) or right (f.minus).
func (f *parserFmt) padString(s string) {
	f.buf.writeString(s)
}

// pad appends b to f.buf, padded on left (!f.minus) or right (f.minus).
func (f *parserFmt) pad(b []byte) {
	f.buf.write(b)
}

// fmtBoolean formats a boolean.
func (f *parserFmt) boolean(v bool) {
	if v {
		f.padString("true")
	} else {
		f.padString("false")
	}
}

// Float formats a float. The default precision for each verb
// is specified as last argument in the call to fmt_float.
// Float formats a float64. It assumes that verb is a valid format specifier
// for strconv.AppendFloat and therefore fits into a byte.
//nolint: unparam
func (f *parserFmt) float(v float64, size int, verb rune, prec int) {
	// Format number, reserving space for leading + sign if needed.
	num := strconv.AppendFloat(f.intbuf[:1], v, byte(verb), prec, size)
	if num[1] == '-' || num[1] == '+' {
		num = num[1:]
	} else {
		num[0] = '+'
	}

	// Special handling for infinities and NaN,
	// which don't look like a number so shouldn't be padded with zeros.
	if num[1] == 'I' || num[1] == 'N' {
		f.pad(num)
		return
	}

	// We want a sign if asked for and if the sign is not positive.
	if f.plus || num[0] != '+' {
		f.pad(num)
		return
	}
	// No sign to show and the number is positive; just print the unsigned number.
	f.pad(num[1:])
}

// complex formats a complex number v with
// r = real(v) and j = imag(v) as (r+ji) using
// fmtFloat for r and j formatting.
func (f *parserFmt) complex(v complex128, size int) {
	oldPlus := f.plus
	f.buf.writeByte('(')
	f.float(real(v), size/2, 'g', -1)
	// Imaginary part always has a sign.
	f.plus = true
	f.float(imag(v), size/2, 'g', -1)
	f.buf.writeString("i)")
	f.plus = oldPlus
}

// fmtInteger formats signed and unsigned integers.
//nolint: unparam
func (f *parserFmt) integer(u uint64, base int, isSigned bool, verb rune, digits string) {
	buf := f.intbuf[0:]
	// Because printing is easier right-to-left: format u into buf, ending at buf[i].
	// We could make things marginally faster by splitting the 32-bit case out
	// into a separate block but it's not worth the duplication, so u has 64 bits.
	i := len(buf)
	for u >= 10 {
		i--
		next := u / 10
		buf[i] = byte('0' + u - next*10)
		u = next
	}
	i--
	buf[i] = digits[u]
	// Left padding with zeros has already been handled like precision earlier
	// or the f.zero flag is ignored due to an explicitly set precision.

	f.pad(buf[i:])
}

// string formats a regular string.
func (f *parserFmt) string(s string) {
	f.padString(s)
}

// bytes formats bytes slice.
func (f *parserFmt) bytes(v []byte) {
	f.buf.writeByte('[')
	for i, c := range v {
		if i > 0 {
			f.buf.writeByte(' ')
		}
		f.integer(uint64(c), 10, unsigned, 'v', sdigits)
	}
	f.buf.writeByte(']')
}

// runes formats runes slice.
func (f *parserFmt) runes(v []rune) {
	for _, c := range v {
		f.buf.writeRune(c)
	}
}
