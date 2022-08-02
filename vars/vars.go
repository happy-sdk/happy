// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	TypeUnknown Type = iota
	TypeString
	TypeBool
	TypeFloat32
	TypeFloat64
	TypeComplex64
	TypeComplex128
	TypeInt
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUint
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeUintptr
	TypeBytes
	TypeRunes
	TypeMap
	TypeReflectVal
	TypeDuration
	TypeArray

	signed   = true
	unsigned = false

	sdigits = "0123456789abcdefx"
	udigits = "0123456789ABCDEFX"

	nilAngleString = "<nil>"
)

//nolint: funlen, cyclop
func (t Type) String() string {
	switch t {
	case TypeUnknown:
		return "unknown"
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	case TypeFloat32:
		return "float32"
	case TypeFloat64:
		return "float64"
	case TypeComplex64:
		return "complex64"
	case TypeComplex128:
		return "complex128"
	case TypeInt:
		return "int"
	case TypeInt8:
		return "int8"
	case TypeInt16:
		return "int16"
	case TypeInt32:
		return "int32"
	case TypeInt64:
		return "int64"
	case TypeUint:
		return "uint"
	case TypeUint8:
		return "uint8"
	case TypeUint16:
		return "uint16"
	case TypeUint32:
		return "uint32"
	case TypeUint64:
		return "uint64"
	case TypeUintptr:
		return "uint64"
	case TypeBytes:
		return "bytes"
	case TypeRunes:
		return "runes"
	case TypeMap:
		return "map"
	case TypeReflectVal:
		return "reflect"
	case TypeDuration:
		return "duration"
	case TypeArray:
		return "array"
	}
	return ""
}

var (

	// ErrVariableKeyEmpty is used when variable key is empty string.
	ErrVariableKeyEmpty = errors.New("variable key can not be empty")

	// EmptyVar variable.
	EmptyVar = Variable{} //nolint: gochecknoglobals

	// parserPool is cached parser.
	//nolint: gochecknoglobals
	parserPool = sync.Pool{
		New: func() interface{} { return new(parser) },
	}
)

type (
	// Type represents type of raw value.
	Type uint

	// Variable is universl representation of key val pair.
	Variable struct {
		key string
		val Value
	}

	// Value describes the variable value.
	Value struct {
		vtype Type
		raw   any
		str   string
	}

	// Collection is like a Go sync.Map safe for concurrent use
	// by multiple goroutines without additional locking or coordination.
	// Loads, stores, and deletes run in amortized constant time.
	//
	// The zero Map is empty and ready for use.
	// A Map must not be copied after first use.
	Collection struct {
		m   sync.Map
		len int64
	}

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

// New return untyped Variable, If error occurred while parsing
// Variable represents default 0, nil value.
func New(key string, val interface{}) Variable {
	return Variable{
		key: key,
		val: NewValue(val),
	}
}

// NewValue return Value, If error occurred while parsing
// VAR represents default 0, nil value.
func NewValue(val any) Value {
	if vv, ok := val.(Value); ok {
		return vv
	}
	p := getParser()
	t, raw := p.printArg(val)
	s := Value{
		vtype: t,
		raw:   raw,
		str:   string(p.buf),
	}
	p.free()
	if t == TypeUnknown && len(s.str) == 0 {
		s.str = fmt.Sprint(s.raw)
	}
	return s
}

// NewFromKeyVal parses variable from single "key=val" pair and
// returns Variable.
func NewFromKeyVal(kv string) (v Variable, err error) {
	if len(kv) == 0 {
		err = ErrVariableKeyEmpty
		return
	}
	reg := regexp.MustCompile(`"([^"]*)"`)

	var key string

	kv = reg.ReplaceAllString(kv, "${1}")
	l := len(kv)
	for i := 0; i < l; i++ {
		if kv[i] == '=' {
			key = kv[:i]
			v = New(key, kv[i+1:])
			if i < l {
				break
			}
		}
	}

	if err == nil && len(key) == 0 {
		err = ErrVariableKeyEmpty
	}
	// VAR did not have any value
	return
}

// NewTyped parses variable and sets appropriately parser error for given type
// if parsing to requested type fails.
func NewTyped(key string, val string, vtype Type) (Variable, error) {
	var variable Variable
	var err error
	value, err := NewTypedValue(val, vtype)
	if err != nil {
		return variable, err
	}
	if len(key) == 0 {
		err = ErrVariableKeyEmpty
	}
	variable = Variable{
		key: key,
		val: value,
	}
	return variable, err
}

// NewTypedValue tries to parse value to given type.
//nolint: cyclop
func NewTypedValue(val string, vtype Type) (Value, error) {
	var v string
	var err error
	var raw interface{}
	if vtype == TypeString {
		return Value{
			str:   val,
			vtype: TypeString,
		}, err
	}
	switch vtype {
	case TypeBool:
		raw, v, err = parseBool(val)
	case TypeFloat32:
		var rawd float64
		rawd, v, err = parseFloat(val, 32)
		raw = float32(rawd)
	case TypeFloat64:
		raw, v, err = parseFloat(val, 64)
	case TypeComplex64:
		raw, v, err = parseComplex64(val)
	case TypeComplex128:
		raw, v, err = parseComplex128(val)
	case TypeInt, TypeInt8, TypeInt16, TypeInt32, TypeInt64:
		raw, v, err = parseInts(val, vtype)
	case TypeUint, TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		raw, v, err = parseUints(val, vtype)
	case TypeUintptr:
		var rawd uint64
		rawd, v, err = parseUint(val, 10, 64)
		raw = uintptr(rawd)
	case TypeBytes:
		raw, v, err = parseBytes(val)
	case TypeDuration:
		raw, err = time.ParseDuration(val)
		// we keep uint64 rep
		v = fmt.Sprintf("%d", raw)
	case TypeMap:
	case TypeReflectVal:
	case TypeRunes:
	case TypeString:
	case TypeUnknown:
		fallthrough
	default:
		v = val
	}

	return Value{
		raw:   raw,
		vtype: vtype,
		str:   v,
	}, err
}

// ParseKeyValSlice parses variables from any []"key=val" slice and
// returns Collection.
func ParseKeyValSlice(kv []string) *Collection {
	vars := new(Collection)
	if len(kv) == 0 {
		return vars
	}
	reg := regexp.MustCompile(`"([^"]*)"`)

NextVar:
	for _, v := range kv {
		v = reg.ReplaceAllString(v, "${1}")
		l := len(v)
		if l == 0 {
			continue
		}
		for i := 0; i < l; i++ {
			if v[i] == '=' {
				vars.Store(v[:i], v[i+1:])
				if i < l {
					continue NextVar
				}
			}
		}
		// VAR did not have any value
		vars.Store(strings.TrimRight(v[:l], "="), "")
	}
	return vars
}

// ParseFromBytes parses []bytes to string, creates []string by new line
// and calls ParseFromStrings.
func ParseFromBytes(b []byte) *Collection {
	slice := strings.Split(string(b[0:]), "\n")
	return ParseKeyValSlice(slice)
}
