// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars

import (
	"errors"
	"sync"
	"time"
)

type (
	// parseBuffer is simple []byte instead of bytes.Buffer to avoid large dependency.
	parserBuffer []byte

	// parser fmt flags placed in a separate struct for easy clearing.
	parserFmtFlags struct {
		plus bool
	}

	// parserFmt is the raw formatter used by Srintf etc.
	// It prints into a buffer that must be set up separately.
	parserFmt struct {
		parserFmtFlags
		buf *parserBuffer // buffer
		// intbuf is large enough to store %b of an int64 with a sign and
		// avoids padding at the end of the struct on 32 bit architectures.
		intbuf [68]byte
	}

	// parser is used to store a printer's state and is reused with
	// sync.Pool to avoid allocations.
	parser struct {
		buf parserBuffer
		// arg holds the current value as an interface{}.
		val any
		// fmt is used to format basic items such as integers or strings.
		fmt parserFmt
		// isCustom is set to true when Ttpe was parsed
		// from custom type.
		isCustom bool
	}
)

const (
	nilAngleString = "<nil>"
	signed         = true
	unsigned       = false
	sdigits        = "0123456789abcdefx"
	udigits        = "0123456789ABCDEFX"
)

var (
	// parserPool is cached parser.
	//nolint: gochecknoglobals
	parserPool = sync.Pool{
		New: func() interface{} { return new(parser) },
	}
)

func ParseKey(str string) (key string, err error) {
	return parseKey(str)
}

func parseBool(str string) (r bool, s string, e error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		r, s = true, "true"
	case "", "0", "f", "F", "false", "FALSE", "False":
		r, s = false, "false"
	default:
		r, s, e = false, "", errorf("%w: can not %s as bool", ErrValueConv, str)
	}
	return r, s, e
}

func parseInts(val string, t Kind) (raw interface{}, v string, err error) {
	var rawd int64
	switch t {
	case KindInt:
		rawd, v, err = parseInt(val, 10, 0)
		raw = int(rawd)
	case KindInt8:
		rawd, v, err = parseInt(val, 10, 8)
		raw = int8(rawd)
	case KindInt16:
		rawd, v, err = parseInt(val, 10, 16)
		raw = int16(rawd)
	case KindInt32:
		rawd, v, err = parseInt(val, 10, 32)
		raw = int32(rawd)
	case KindInt64:
		raw, v, err = parseInt(val, 10, 64)
	}
	return
}

func parseInt(str string, base, bitSize int) (r int64, s string, err error) {
	if str == "true" {
		return 1, "1", nil
	}
	if str == "false" {
		return 0, "0", nil
	}
	r, e := parseIntFast(str, base, bitSize)
	if e != nil {
		err = errors.Join(ErrValueConv, e)
	} else {
		s = formatIntFast(r, 10)
	}
	return r, s, err
}

func parseUints(val string, t Kind) (raw interface{}, v string, err error) {
	var rawd uint64
	switch t {
	case KindUint:
		rawd, v, err = parseUint(val, 10, 0)
		raw = uint(rawd)
	case KindUint8:
		rawd, v, err = parseUint(val, 10, 8)
		raw = uint8(rawd)
	case KindUint16:
		rawd, v, err = parseUint(val, 10, 16)
		raw = uint16(rawd)
	case KindUint32:
		rawd, v, err = parseUint(val, 10, 32)
		raw = uint32(rawd)
	case KindUint64:
		raw, v, err = parseUint(val, 10, 64)
	}

	return
}

func parseUint(str string, base, bitSize int) (r uint64, s string, err error) {
	if str == "true" {
		return 1, "1", nil
	}
	if str == "false" {
		return 0, "0", nil
	}
	r, e := strconvParseUint(str, base, bitSize)
	if e != nil {
		err = errors.Join(ErrValueConv, e)
	} else {
		s = formatUintFast(r, base)
	}
	return r, s, err
}

func parseFloat(str string, bitSize int) (r float64, s string, err error) {
	if str == "true" {
		return 1, "1", nil
	}
	if str == "false" {
		return 0, "0", nil
	}
	r, e := parseFloatFast(str, bitSize)
	if e != nil {
		err = errors.Join(ErrValueConv, e)
	} else {
		s = string(fastFtoa(make([]byte, 0, 24), r, 'g', -1, bitSize))
	}
	return r, s, err
}

func parseComplex64(str string) (r complex64, s string, e error) {
	if str == "true" {
		str = "1"
	}
	if str == "false" {
		str = "0"
	}
	if c, err := parseComplexFast(str, 64); err == nil {
		return complex64(c), str, nil
	}
	fields := stringsFields(str)
	if len(fields) != 2 {
		return complex64(0), "", errorf("%w: %s can not parsed as complex64", ErrValueConv, str)
	}
	var err error
	var f1, f2 float32
	var s1, s2 string
	lf1, s1, err := parseFloat(fields[0], 32)
	if err != nil {
		return complex64(0), "", err
	}
	f1 = float32(lf1)

	rf2, s2, err := parseFloat(fields[1], 32)
	if err != nil {
		return complex64(0), "", err
	}
	f2 = float32(rf2)
	s = s1 + " " + s2
	r = complex(f1, f2)
	return r, s, e
}

func parseComplex128(str string) (r complex128, s string, e error) {
	if str == "true" {
		str = "1"
	}
	if str == "false" {
		str = "0"
	}
	if c, err := parseComplexFast(str, 128); err == nil {
		return c, str, nil
	}
	fields := stringsFields(str)
	if len(fields) != 2 {
		return complex128(0), "", errorf("%w: %s can not parsed as complex64", ErrValueConv, str)
	}
	var err error
	var f1, f2 float64
	var s1, s2 string
	lf1, s1, err := parseFloat(fields[0], 64)
	if err != nil {
		return complex128(0), "", err
	}
	f1 = lf1

	rf2, s2, err := parseFloat(fields[1], 64)
	if err != nil {
		return complex128(0), "", err
	}
	f2 = rf2
	s = s1 + " " + s2
	r = complex(f1, f2)
	return r, s, e
}

// getParser allocates a new parser struct or grabs a cached one.
func getParser() (p *parser) {
	p, _ = parserPool.Get().(*parser)
	p.isCustom = false
	p.fmt.init(&p.buf)
	return p
}

func (f *parserFmt) init(buf *parserBuffer) {
	f.buf = buf
	f.parserFmtFlags = parserFmtFlags{plus: false}
}

// free saves used parser structs in parserPool;
// that avoids an allocation per invocation.
func (p *parser) free() {
	// Proper usage of a sync.Pool requires each entry to have approximately
	// the same memory cost. To obtain this property when the stored type
	// contains a variably-sized buffer, we add a hard limit on the maximum buffer
	// to place back in the pool.
	//
	// See https://golang.org/issue/23199
	if cap(p.buf) < 64<<10 {
		p.buf = p.buf[:0]
		p.val = nil
		parserPool.Put(p)
	}
}

func (p *parser) parseValue(val any) (typ Kind, err error) {
	p.val = val

	if val == nil {
		p.fmt.string(nilAngleString)
		kind := KindOf(val)
		if kind == KindInvalid {
			err = errorf("%w: %#v", ErrValueInvalid, val)
		}
		return kind, err
	}

	if v, ok := val.(stringer); ok {
		typ = KindString
		p.fmt.string(v.String())
		return typ, nil
	}

	switch v := val.(type) {
	case bool:
		typ = KindBool
		p.fmt.boolean(v)
	case int:
		typ = KindInt
		p.fmt.integer(uint64(v), signed, sdigits)
	case int8:
		typ = KindInt8
		p.fmt.integer(uint64(v), signed, sdigits)
	case int16:
		typ = KindInt16
		p.fmt.integer(uint64(v), signed, sdigits)
	case int32:
		typ = KindInt32
		p.fmt.integer(uint64(v), signed, sdigits)
	case int64:
		typ = KindInt64
		p.fmt.integer(uint64(v), signed, sdigits)
	case uint:
		typ = KindUint
		p.fmt.integer(uint64(v), unsigned, udigits)
	case uint8:
		typ = KindUint8
		p.fmt.integer(uint64(v), unsigned, udigits)
	case uint16:
		typ = KindUint16
		p.fmt.integer(uint64(v), unsigned, udigits)
	case uint32:
		typ = KindUint32
		p.fmt.integer(uint64(v), unsigned, udigits)
	case uint64:
		typ = KindUint64
		p.fmt.integer(v, unsigned, udigits)
	case uintptr:
		typ = KindUintptr
		p.fmt.integer(uint64(v), unsigned, udigits)
	case float32:
		typ = KindFloat32
		p.fmt.float(float64(v), 32, 'g', -1)
	case float64:
		typ = KindFloat64
		p.fmt.float(v, 64, 'g', -1)
	case complex64:
		typ = KindComplex64
		p.fmt.complex(complex128(v), 64)
	case complex128:
		typ = KindComplex128
		p.fmt.complex(v, 128)
	case string:
		typ = KindString
		p.fmt.string(v)
	case time.Duration:
		typ = KindDuration
		p.fmt.string(v.String())
	default:
		typ, err = p.parseUnderlyingAsKind(val)
	}
	return typ, err
}

func (p *parser) parseUnderlyingAsKindFromPointer(val any) (Kind, error) {
	var (
		underlying   any
		localtype    Kind
		implStringer bool
	)
	// if _, ok := val.(stringer); val != nil && ok {
	// 	// p.fmt.string(vv.String())
	// 	implStringer = true
	// }

	switch v := val.(type) {
	case *bool:
		localtype = KindBool
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.boolean(*v)
			}
		}
	case *int:
		localtype = KindInt
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), signed, sdigits)
			}
		}
	case *int8:
		localtype = KindInt8
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), signed, sdigits)
			}
		}
	case *int16:
		localtype = KindInt16
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), signed, sdigits)
			}
		}
	case *int32:
		localtype = KindInt32
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), signed, sdigits)
			}
		}
	case *int64:
		localtype = KindInt64
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), signed, sdigits)
			}
		}
	case *uint:
		localtype = KindUint
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), unsigned, udigits)
			}
		}
	case *uint8:
		localtype = KindUint8
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), unsigned, udigits)
			}
		}
	case *uint16:
		localtype = KindUint16
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), unsigned, udigits)
			}
		}
	case *uint32:
		localtype = KindUint32
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), unsigned, udigits)
			}
		}
	case *uint64:
		localtype = KindUint64
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), unsigned, udigits)
			}
		}
	case *uintptr:
		localtype = KindUintptr
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.integer(uint64(*v), unsigned, udigits)
			}
		}
	case *float32:
		localtype = KindFloat32
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.float(float64(*v), 32, 'g', -1)
			}
		}
	case *float64:
		localtype = KindFloat64
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.float(*v, 64, 'g', -1)
			}
		}
	case *complex64:
		localtype = KindComplex64
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.complex(complex128(*v), 64)
			}
		}
	case *complex128:
		localtype = KindComplex128
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.complex(*v, 128)
			}
		}
	case *string:
		localtype = KindString
		if v != nil {
			underlying = *v
			if !implStringer {
				p.fmt.string(*v)
			}
		}
	default:
		if v == nil {
			underlying = "" // use empty string for nil ptr
			localtype = KindInvalid
		}
	}
	p.val = underlying
	return localtype, nil
}

// parseUnderlyingAsKind is unsafe function.
// it takes non builtin arg and to parses it to given Kind.
// Before calling you must be sure that val can be casted into Kind.
func (p *parser) parseUnderlyingAsKind(val any) (Kind, error) {
	// check is it pointer
	pval, typ := underlyingValueOf(val, true)
	// first check does type implment stringer.
	// so that we can write string representation of value
	// to buffer directly.
	var implStringer bool
	// unsafe ptr with nil value would satisfy stringer
	// so check that value is actually present
	if str, ok := val.(stringer); pval != nil && ok {
		p.fmt.string(str.String())
		implStringer = true
	}

	var (
		underlying any
		localtype  Kind
	)

	if pval == nil {
		return typ, nil
	}

	if typ == KindPointer {
		return p.parseUnderlyingAsKindFromPointer(val)
	}

	switch v := pval.(type) {
	case bool:
		underlying = v
		localtype = KindBool
		if !implStringer {
			p.fmt.boolean(v)
		}
	case int:
		underlying = v
		localtype = KindInt
		if !implStringer {
			p.fmt.integer(uint64(v), signed, sdigits)
		}
	case int8:
		underlying = v
		localtype = KindInt8
		if !implStringer {
			p.fmt.integer(uint64(v), signed, sdigits)
		}
	case int16:
		underlying = v
		localtype = KindInt16
		if !implStringer {
			p.fmt.integer(uint64(v), signed, sdigits)
		}
	case int32:
		underlying = v
		localtype = KindInt32
		if !implStringer {
			p.fmt.integer(uint64(v), signed, sdigits)
		}
	case int64:
		underlying = v
		localtype = KindInt64
		if !implStringer {
			p.fmt.integer(uint64(v), signed, sdigits)
		}
	case uint:
		underlying = v
		localtype = KindUint
		if !implStringer {
			p.fmt.integer(uint64(v), unsigned, udigits)
		}
	case uint8:
		underlying = v
		localtype = KindUint8
		if !implStringer {
			p.fmt.integer(uint64(v), unsigned, udigits)
		}
	case uint16:
		underlying = v
		localtype = KindUint16
		if !implStringer {
			p.fmt.integer(uint64(v), unsigned, udigits)
		}
	case uint32:
		underlying = v
		localtype = KindUint32
		if !implStringer {
			p.fmt.integer(uint64(v), unsigned, udigits)
		}
	case uint64:
		underlying = v
		localtype = KindUint64
		if !implStringer {
			p.fmt.integer(uint64(v), unsigned, udigits)
		}
	case uintptr:
		underlying = v
		localtype = KindUintptr
		if !implStringer {
			p.fmt.integer(uint64(v), unsigned, udigits)
		}
	case float32:
		underlying = v
		localtype = KindFloat32
		if !implStringer {
			p.fmt.float(float64(v), 32, 'g', -1)
		}
	case float64:
		underlying = v
		localtype = KindFloat64
		if !implStringer {
			p.fmt.float(v, 64, 'g', -1)
		}
	case complex64:
		underlying = v
		localtype = KindComplex64
		if !implStringer {
			p.fmt.complex(complex128(v), 64)
		}
	case complex128:
		underlying = v
		localtype = KindComplex128
		if !implStringer {
			p.fmt.complex(v, 128)
		}
	case string:
		underlying = v
		localtype = KindString
		if !implStringer {
			p.fmt.string(v)
		}
	case []byte:
		underlying = v
		localtype = KindSlice
		if !implStringer {
			p.fmt.write(v)
		}
	default:
		return typ,
			errorf("%w: type parser to parse %s is not implemented",
				ErrValue,
				typ.String(),
			)
	}
	// we mark it custom only if value
	// is implements Stringerr so we know that
	// most likely we can not other values from string represenation.
	p.isCustom = implStringer
	// assign that raw custom value as builtin
	p.val = underlying
	return localtype, nil
}

// string appends s to f.buf,
// and formats a regular string.
func (f *parserFmt) string(s string) {
	f.buf.writeString(s)
}

// boolean formats a boolean.
func (f *parserFmt) boolean(v bool) {
	if v {
		f.string("true")
	} else {
		f.string("false")
	}
}

// integer formats signed and unsigned integers.
func (f *parserFmt) integer(u uint64, isSigned bool, digits string) {
	negative := isSigned && int64(u) < 0
	if negative {
		u = -u
	}

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
	if negative {
		i--
		buf[i] = '-'
	}
	f.buf.write(buf[i:])
}

// Float formats a float. The default precision for each verb
// is specified as last argument in the call to fmt_float.
// Float formats a float64. It assumes that verb is a valid format specifier
// for strconv.AppendFloat and therefore fits into a byte.
// nolint: unparam
func (f *parserFmt) float(v float64, size int, verb rune, prec int) {
	// Format number, reserving space for leading + sign if needed.
	num := fastFtoa(f.intbuf[:1], v, byte(verb), prec, size)
	if num[1] == '-' || num[1] == '+' {
		num = num[1:]
	} else {
		num[0] = '+'
	}

	// Special handling for infinities and NaN,
	// which don't look like a number so shouldn't be padded with zeros.
	if num[1] == 'N' {
		f.write(num[1:])
		return
	}
	if num[1] == 'I' {
		f.write(num)
		return
	}

	// We want a sign if asked for and if the sign is not positive.
	if f.plus || num[0] != '+' {
		f.write(num)
		return
	}
	// No sign to show and the number is positive; just print the unsigned number.
	f.write(num[1:])
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

// pad appends b to f.buf, padded on left (!f.minus) or right (f.minus).
func (f *parserFmt) write(b []byte) {
	f.buf.write(b)
}

// parserBuffer
func (b *parserBuffer) write(p []byte) {
	*b = append(*b, p...)
}

func (b *parserBuffer) writeString(s string) {
	*b = append(*b, s...)
}

func (b *parserBuffer) writeByte(c byte) {
	*b = append(*b, c)
}

func normalizeValue(str string) string {
	str = nfc.String(str)
	str = stringsTrimSpace(str)
	// Check if the string is surrounded by quotes
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	return str
}

// Is reports whether the rune is in the specified table of ranges.

// IsSpace reports whether the rune is a space character as defined
// by Unicode's White Space property; in the Latin-1 space
// this is
//
//	'\t', '\n', '\v', '\f', '\r', ' ', U+0085 (NEL), U+00A0 (NBSP).
//
// Other definitions of spacing characters are set by category
// Z and property Pattern_White_Space.
func unicodeIsSpace(r rune) bool {
	// This property isn't the same as Z; special-case it.
	if uint32(r) <= unicodeMaxLatin1 {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
			return true
		}
		return false
	}
	return unicodeIsExcludingLatin(unicodeWhiteSpace, r)
}
