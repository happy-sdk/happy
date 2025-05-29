// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package vars provides the API to parse variables from various input
// formats/kinds to common key value pair. Or key value pair sets to Variable collections.
package vars

import (
	"errors"
	"fmt"
	"time"
)

var (
	EmptyVariable = Variable{}
	EmptyValue    = Value{}
	NilValue      = Value{
		kind: KindInvalid,
		str:  "nil",
	}
)

var (
	ErrReadOnly = errors.New("vars.readonly")

	// Key errors
	ErrKey                      = errors.New("key error")
	ErrKeyPrefix                = fmt.Errorf("%w: key can not start with [0-9]", ErrKey)
	ErrKeyIsEmpty               = fmt.Errorf("%w: key was empty string", ErrKey)
	ErrKeyHasIllegalChar        = fmt.Errorf("%w: key has illegal characters", ErrKey)
	ErrKeyHasIllegalStarterByte = fmt.Errorf("%w: key has illegal starter byte", ErrKey)
	ErrKeyHasControlChar        = fmt.Errorf("%w: key contains some of unicode control character(s)", ErrKey)
	ErrKeyNotValidUTF8          = fmt.Errorf("%w: provided key was not valid UTF-8 string", ErrKey)
	ErrKeyHasNonPrintChar       = fmt.Errorf("%w: key contains some of non print character(s)", ErrKey)
	ErrKeyOutOfRange            = fmt.Errorf("%w: key contained utf8 character out of allowed range", ErrKey)

	// Value errors
	ErrValue        = errors.New("value error")
	ErrValueInvalid = fmt.Errorf("%w: invalid value", ErrValue)
	ErrValueConv    = fmt.Errorf("%w: failed to convert value", ErrValue)
	// Parser errors

	// ErrRange indicates that a value is out of range for the target type.
	ErrRange = fmt.Errorf("%w: value out of range", ErrValue)
	// ErrSyntax indicates that a value does not have the right syntax for the target type.
	ErrSyntax = fmt.Errorf("%w: invalid syntax", ErrValue)
)

// KindOf returns kind for provided  value.
func KindOf(in any) (kind Kind) {
	_, kind = underlyingValueOf(in, false)
	return
}

// New parses key and value into Variable.
// Error is returned when parsing of key or value fails.
func New(name string, val any, ro bool) (Variable, error) {
	name, err := parseKey(name)
	if err != nil {
		return EmptyVariable, err
	}
	v, err := NewValue(val)
	return Variable{
		name: name,
		val:  v,
		ro:   ro,
	}, err
}

func String(key string, val string) Variable {
	return Variable{
		name: key,
		val: Value{
			raw: val,
			str: val,
		},
	}
}

func StringValue(val string) Value {
	return Value{
		raw: val,
		str: val,
	}
}

func NewAs(name string, val any, ro bool, kind Kind) (Variable, error) {
	v, err := NewValueAs(val, kind)
	if err != nil {
		return EmptyVariable, err
	}
	return New(name, v, ro)
}

func EmptyNamedVariable(name string) (Variable, error) {
	v := EmptyVariable
	name, err := parseKey(name)
	if err != nil {
		return EmptyVariable, err
	}
	v.name = name
	return v, nil
}

// ParseVariableFromString parses variable from single key=val pair and returns a Variable
// if parsing is successful. EmptyVariable and error is returned when parsing fails.
func ParseVariableFromString(kv string) (Variable, error) {
	if len(kv) == 0 {
		return EmptyVariable, ErrKey
	}
	k, v, _ := stringsCut(kv, '=')

	key, err := parseKey(k)
	if err != nil {
		return EmptyVariable, fmt.Errorf("%w: failed to parse variable key", err)
	}

	return New(key, normalizeValue(v), false)
}

// NewValue parses provided val into Value
// Error is returned if parsing fails.
func NewValue(val any) (Value, error) {

	if vv, ok := val.(Value); ok {
		if vv.kind == KindInvalid {
			return EmptyValue, fmt.Errorf("%w: %#v", ErrValueInvalid, val)
		}
		return vv, nil
	}

	if vv, ok := val.(Variable); ok {
		if vv.Kind() == KindInvalid {
			return EmptyValue, fmt.Errorf("%w: variable value %#v", ErrValueInvalid, val)
		}
		return vv.val, nil
	}
	p := getParser()
	defer p.free()

	kind, err := p.parseValue(val)
	v := Value{
		raw:      p.val,
		kind:     kind,
		str:      string(p.buf),
		isCustom: p.isCustom,
	}
	return v, err
}

func NewValueAs(val any, kind Kind) (Value, error) {
	p := getParser()
	akind, err := p.parseValue(val)
	if err != nil {
		p.free()
		return EmptyValue, err
	}
	if kind == akind {
		defer p.free()
		return Value{
			raw:      p.val,
			kind:     kind,
			str:      string(p.buf),
			isCustom: p.isCustom,
		}, nil
	}

	str := string(p.buf)
	p.free()

	if v, err := convert(val, akind, kind); err == nil {
		return v, nil
	}

	return ParseValueAs(str, kind)
}

func convertInt64(val int64, to Kind) (Value, error) {
	v := Value{
		kind: to,
	}
	switch to {
	case KindBool:
		switch val {
		case 0:
			v.raw = false
		case 1:
			v.raw = true
		}
	case KindInt:
		v.raw = int(val)
	case KindInt8:
		v.raw = int8(val)
	case KindInt16:
		v.raw = int16(val)
	case KindInt32:
		v.raw = int32(val)
	case KindInt64:
		v.raw = val
	case KindUint:
		v.raw = uint(val)
	case KindUint8:
		v.raw = uint8(val)
	case KindUint16:
		v.raw = uint16(val)
	case KindUint32:
		v.raw = uint32(val)
	case KindUint64:
		v.raw = uint64(val)
	case KindUintptr:
		v.raw = uintptr(val)
	case KindFloat32:
		v.raw = float32(val)
	case KindFloat64:
		v.raw = float64(val)
	case KindComplex64:
		v.raw = complex64(complex(float64(val), 0))
	case KindComplex128:
		v.raw = complex(float64(val), 0)
	}
	if v.raw != nil {
		return v, nil
	}
	return EmptyValue, fmt.Errorf("%w: %d to %s", ErrValueConv, val, to.String())
}

func convertUint64(val uint64, to Kind) (Value, error) {
	v := Value{
		kind: to,
	}
	switch to {
	case KindBool:
		switch val {
		case 0:
			v.raw = false
		case 1:
			v.raw = true
		}
	case KindInt:
		v.raw = int(val)
	case KindInt8:
		v.raw = int8(val)
	case KindInt16:
		v.raw = int16(val)
	case KindInt32:
		v.raw = int32(val)
	case KindInt64:
		v.raw = val
	case KindUint:
		v.raw = uint(val)
	case KindUint8:
		v.raw = uint8(val)
	case KindUint16:
		v.raw = uint16(val)
	case KindUint32:
		v.raw = uint32(val)
	case KindUint64:
		v.raw = uint64(val)
	case KindUintptr:
		v.raw = uintptr(val)
	case KindFloat32:
		v.raw = float32(val)
	case KindFloat64:
		v.raw = float64(val)
	case KindComplex64:
		v.raw = complex64(complex(float64(val), 0))
	case KindComplex128:
		v.raw = complex(float64(val), 0)
	}
	if v.raw != nil {
		return v, nil
	}
	return EmptyValue, fmt.Errorf("%w: %d to %s", ErrValueConv, val, to.String())
}

func convertFloat64(val float64, to Kind) (Value, error) {
	v := Value{
		kind: to,
	}
	switch to {
	case KindBool:
		switch val {
		case 0:
			v.raw = false
		case 1:
			v.raw = true
		}
	case KindInt:
		v.raw = int(val)
	case KindInt8:
		v.raw = int8(val)
	case KindInt16:
		v.raw = int16(val)
	case KindInt32:
		v.raw = int32(val)
	case KindInt64:
		v.raw = val
	case KindUint:
		v.raw = uint(val)
	case KindUint8:
		v.raw = uint8(val)
	case KindUint16:
		v.raw = uint16(val)
	case KindUint32:
		v.raw = uint32(val)
	case KindUint64:
		v.raw = uint64(val)
	case KindUintptr:
		v.raw = uintptr(val)
	case KindFloat32:
		v.raw = float32(val)
	case KindFloat64:
		v.raw = float64(val)
	case KindComplex64:
		v.raw = complex64(complex(float64(val), 0))
	case KindComplex128:
		v.raw = complex(float64(val), 0)
	}
	if v.raw != nil {
		return v, nil
	}
	return EmptyValue, fmt.Errorf("%w: %f to %s", ErrValueConv, val, to.String())
}

func convert(raw any, from, to Kind) (Value, error) {
	p := getParser()
	defer p.free()

	if from >= KindInt && from <= KindInt64 {
		val, ok := raw.(int64)
		if ok {
			if v, err := convertInt64(val, to); err == nil {
				if _, err := p.parseValue(v); err != nil {
					return EmptyValue, err
				}
				v.str = string(p.buf)
				return v, nil
			}
		}
	} else if from >= KindUint && from <= KindUintptr {
		val, ok := raw.(uint64)
		if ok {
			if v, err := convertUint64(val, to); err == nil {
				if _, err := p.parseValue(v); err != nil {
					return EmptyValue, err
				}
				v.str = string(p.buf)
				return v, nil
			}
		}
	} else if from == KindFloat32 || from == KindFloat64 {
		val, ok := raw.(float64)
		if ok {
			if v, err := convertFloat64(val, to); err == nil {
				if _, err := p.parseValue(v); err != nil {
					return EmptyValue, err
				}
				v.str = string(p.buf)
				return v, nil
			}
		}
	} else if to == KindDuration && from == KindString {
		val, ok := raw.(string)
		if ok {
			v := Value{}
			v.kind = KindDuration
			d, err := time.ParseDuration(val)
			if err != nil {
				return EmptyValue, err
			}
			v.str = val
			v.raw = d
			return v, nil
		}
	}

	return EmptyValue, fmt.Errorf("%w: %v to %s", ErrValueConv, raw, to.String())
}

func ParseValueAs(val string, kind Kind) (Value, error) {
	if kind == KindString {
		return Value{
			kind: KindString,
			str:  val,
			raw:  val,
		}, nil
	}
	var str string
	var err error
	var raw any
	switch kind {
	case KindBool:
		raw, str, err = parseBool(val)
	case KindFloat32:
		var rawd float64
		rawd, str, err = parseFloat(val, 32)
		raw = float32(rawd)
	case KindFloat64:
		raw, str, err = parseFloat(val, 64)
	case KindComplex64:
		raw, str, err = parseComplex64(val)
	case KindComplex128:
		raw, str, err = parseComplex128(val)
	case KindInt, KindInt8, KindInt16, KindInt32, KindInt64:
		raw, str, err = parseInts(val, kind)
	case KindUint, KindUint8, KindUint16, KindUint32, KindUint64:
		raw, str, err = parseUints(val, kind)
	case KindUintptr:
		var rawd uint64
		rawd, str, err = parseUint(val, 10, 64)
		raw = uintptr(rawd)
	case KindSlice:
		raw, str = val, val
	default:
		err = fmt.Errorf("%w: can not create kind value %s from %s", ErrValue, kind.String(), val)
	}

	if err != nil {
		err = fmt.Errorf("%w: can not parse %s as %s", err, val, kind.String())
		kind = KindInvalid
	}

	return Value{
		raw:  raw,
		kind: kind,
		str:  str,
	}, err
}

// ParseKinddVariable parses variable and returns parser error for given kind
// if parsing to requested kind fails.
func ParseVariableAs(key, val string, ro bool, kind Kind) (Variable, error) {
	v, err := ParseValueAs(val, kind)
	if err != nil {
		return EmptyVariable, err
	}
	return New(key, v, ro)
}

func AsVariable[VAR VariableIface[VAL], VAL ValueIface](in Variable) VAR {
	var v VariableIface[VAL] = GenericVariable[VAL]{
		ro:   in.ReadOnly(),
		name: in.Name(),
		val:  in.Value(),
	}
	return v.(VAR)
}

func ValueOf(val any) Value {
	v, _ := NewValue(val)
	return v
}

// ParseFromBytes parses []bytes to string, creates []string by new line
// and calls ParseFromStrings.
func ParseMapFromBytes(b []byte) (*Map, error) {
	slice := stringsSplit(string(b[0:]), '\n')
	return ParseMapFromSlice(slice)
}

// ParseKeyValSlice parses variables from any []"key=val" slice and
// returns Collection.
func ParseMapFromSlice(kv []string) (*Map, error) {
	vars := new(Map)

	if len(kv) == 0 {
		return vars, nil
	}

	for _, v := range kv {
		// allow empty lines
		if len(v) == 0 {
			continue
		}
		vv, err := ParseVariableFromString(v)
		if err != nil {
			return nil, err
		}
		if err := vars.Store(vv.Name(), vv.Value()); err != nil {
			return nil, err
		}
	}
	return vars, nil
}

func ParseMapFromGoMap(m map[string]string) (*Map, error) {
	mm := new(Map)

	if m == nil {
		return mm, nil
	}

	for k, v := range m {
		if err := mm.Store(k, v); err != nil {
			return nil, err
		}
	}

	return mm, nil
}

func errorf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}
