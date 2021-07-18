// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"errors"
	"regexp"
	"sync"
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

	signed   = true
	unsigned = false

	sdigits = "0123456789abcdefx"
	udigits = "0123456789ABCDEFX"

	nilAngleString = "<nil>"
)

var (
	// parserPool is cached parser
	parserPool = sync.Pool{
		New: func() interface{} { return new(parser) },
	}

	// ErrVariableKeyEmpty is used when variable key is empty string
	ErrVariableKeyEmpty = errors.New("variable key can not be empty")
)

type (
	// Type represents type of raw value
	Type uint

	// Variable is universl representation of key val pair
	Variable struct {
		key string
		val Value
	}

	// Value describes the variable value
	Value struct {
		vtype Type
		raw   interface{}
		str   string
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
// Variable represents default 0, nil value
func New(key string, val interface{}) (Variable, error) {
	v, err := NewValue(val)
	vv := Variable{
		key: key,
		val: v,
	}
	return vv, err
}

// NewValue return Value, If error occurred while parsing
// VAR represents default 0, nil value
func NewValue(val interface{}) (Value, error) {
	p := getParser()
	t, raw := p.printArg(val)
	s := Value{
		vtype: t,
		raw:   raw,
		str:   string(p.buf),
	}
	p.free()
	return s, nil
}

// NewFromKeyVal parses variable from single "key=val" pair and
// returns Variable
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
			v, err = New(key, kv[i+1:])
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
// if parsing to requested type fails
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

// NewTypedValue tries to parse value to given type
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
		raw, v, err = parseFloat(val, 32)
		raw = float32(raw.(float64))
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
		raw, v, err = parseUint(val, 10, 64)
		raw = uintptr(raw.(uint64))
	case TypeBytes:
		raw, v, err = parseBytes(val)
	}

	return Value{
		raw:   raw,
		vtype: vtype,
		str:   v,
	}, err
}
