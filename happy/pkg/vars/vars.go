// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Package vars provides the API to parse variables from various input
// formats/kinds to common key value pair.
// Or key value pair sets to Variable collections.
package vars

import (
	"errors"
	"fmt"
)

var (
	EmptyVariable = Variable{}
	EmptyValue    = Value{}
)

var (
	ErrReadOnly = errors.New("readonly")

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
	src, err := NewValue(val)
	if err != nil {
		return EmptyValue, err
	}
	if src.Kind() == kind {
		return src, nil
	}
	return ParseValueAs(src.String(), kind)
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
	var v VariableIface[VAL]
	v = GenericVariable[VAL]{
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
		vars.Store(vv.Name(), vv.Value())
	}
	return vars, nil
}

func errorf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}
