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
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	signed   = true
	unsigned = false

	sdigits = "0123456789abcdefx"
	udigits = "0123456789ABCDEFX"

	nilAngleString = "<nil>"
)

var (

	// parserPool is cached parser.
	//nolint: gochecknoglobals
	parserPool = sync.Pool{
		New: func() interface{} { return new(parser) },
	}
)

type (

	// parser is used to store a printer's state and is reused with
	// sync.Pool to avoid allocations.
	parser struct {
		buf parserBuffer
		// arg holds the current value as an interface{}.
		arg interface{}
		// erroring is set when printing an error
		// string to guard against calling handleMethods.
		erroring bool
		// fmt is used to format basic items such as integers or strings.
		fmt parserFmt
	}

	// parserFmt is the raw formatter used by Printf etc.
	// It prints into a buffer that must be set up separately.
	parserFmt struct {
		parserFmtFlags
		buf *parserBuffer // buffer
		// intbuf is large enough to store %b of an int64 with a sign and
		// avoids padding at the end of the struct on 32 bit architectures.
		intbuf [68]byte
	}

	// parseBuffer is simple []byte instead of bytes.Buffer to avoid large dependency.
	parserBuffer []byte

	// parser fmt flags placed in a separate struct for easy clearing.
	parserFmtFlags struct {
		plus bool
	}
)

// getParser allocates a new parser struct or grabs a cached one.
func getParser() (p *parser) {
	p, _ = parserPool.Get().(*parser)
	p.erroring = false
	p.fmt.init(&p.buf)
	return p
}

// Flag satisfies fmt.State.
func (p *parser) Flag(b int) bool {
	return false
}

// Precision satisfies fmt.State.
func (p *parser) Precision() (prec int, ok bool) { return 0, false }

// Width satisfies fmt.State.
func (p *parser) Width() (wid int, ok bool) { return 0, false }

// Write satisfies fmt.State.
func (p *parser) Write(b []byte) (ret int, err error) { return 0, nil }

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
		p.arg = nil
		parserPool.Put(p)
	}
}

func (p *parser) parseArg(arg any) (vtype Type, raw any) {
	p.arg = arg
	if arg == nil {
		p.fmt.padString(nilAngleString)
		return
	}
	// Some types can be done without reflection.
	switch f := arg.(type) {

	case bool:
		p.fmt.boolean(f)
		vtype = TypeBool
	case float32:
		p.fmt.float(float64(f), 32, 'g', -1)
		vtype = TypeFloat32
	case float64:
		p.fmt.float(f, 64, 'g', -1)
		vtype = TypeFloat64
	case complex64:
		p.fmt.complex(complex128(f), 64)
		vtype = TypeComplex64
	case complex128:
		p.fmt.complex(f, 128)
		vtype = TypeComplex128
	case int:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt
	case int8:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt8
	case int16:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt16
	case int32:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt32
	case int64:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt64
	case uint:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint
	case uint8:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint8
	case uint16:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint16
	case uint32:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint32
	case uint64:

		p.fmt.integer(f, 10, unsigned, 'v', udigits)
		vtype = TypeUint64
	case uintptr:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUintptr
	case string:
		p.fmt.string(f)
		vtype = TypeString
	case []byte:
		p.fmt.bytes(f)
		vtype = TypeBytes
	case []rune:
		p.fmt.runes(f)
		vtype = TypeRunes
	default:
		// If the type is not simple, it might have methods.
		vtype = TypeUnknown
	}
	return vtype, arg
}

func parseBool(str string) (r bool, s string, e error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		r, s = true, "true"
	case "0", "f", "F", "false", "FALSE", "False":
		r, s = false, "false"
	default:
		r, s, e = false, "", strconv.ErrSyntax
	}
	return r, s, e
}

func parseFloat(str string, bitSize int) (r float64, s string, e error) {
	r, e = strconv.ParseFloat(str, bitSize)
	if bitSize == 32 {
		s = fmt.Sprintf("%v", float32(r))
	} else {
		s = fmt.Sprintf("%v", r)
	}
	return r, s, e
}

func parseComplex64(str string) (r complex64, s string, e error) {
	fields := strings.Fields(str)
	if len(fields) != 2 {
		return complex64(0), "", strconv.ErrSyntax
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
	fields := strings.Fields(str)
	if len(fields) != 2 {
		return complex128(0), "", strconv.ErrSyntax
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

func parseInts(val string, vtype Type) (raw interface{}, v string, err error) {
	var rawd int64
	switch vtype {
	case TypeInt:
		rawd, v, err = parseInt(val, 10, 0)
		raw = int(rawd)
	case TypeInt8:
		rawd, v, err = parseInt(val, 10, 8)
		raw = int8(rawd)
	case TypeInt16:
		rawd, v, err = parseInt(val, 10, 16)
		raw = int16(rawd)
	case TypeInt32:
		rawd, v, err = parseInt(val, 10, 32)
		raw = int32(rawd)
	case TypeInt64:
		raw, v, err = parseInt(val, 10, 64)
	}
	return
}

func parseInt(str string, base, bitSize int) (r int64, s string, e error) {
	r, e = strconv.ParseInt(str, base, bitSize)
	s = strconv.Itoa(int(r))
	return r, s, e
}

func parseUints(val string, vtype Type) (raw interface{}, v string, err error) {
	var rawd uint64
	switch vtype {
	case TypeUint:
		rawd, v, err = parseUint(val, 10, 0)
		raw = uint(rawd)
	case TypeUint8:
		rawd, v, err = parseUint(val, 10, 8)
		raw = uint8(rawd)
	case TypeUint16:
		rawd, v, err = parseUint(val, 10, 16)
		raw = uint16(rawd)
	case TypeUint32:
		rawd, v, err = parseUint(val, 10, 32)
		raw = uint32(rawd)
	case TypeUint64:
		raw, v, err = parseUint(val, 10, 64)
	}

	return
}

func parseUint(str string, base, bitSize int) (r uint64, s string, e error) {
	r, e = strconv.ParseUint(str, base, bitSize)
	s = strconv.FormatUint(r, base)
	return r, s, e
}

func parseBytes(str string) (b []byte, s string, e error) {
	str = strings.TrimLeft(str, "[")
	str = strings.TrimRight(str, "]")
	fields := strings.Fields(str)
	for _, c := range fields {
		raw, _, err := parseUint(c, 10, 8)
		if err != nil {
			return b, str, err
		}
		b = append(b, uint8(raw))
	}
	return b, str, nil
}
