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

// Package vars provides the API to parse variables from various input
// formats/types to common key value pair Variable or sets to Collection.
// Variable = (k string, v Value)
// Collection is sync map like collection of Variables
//
// Main purpose of this library is to provide simple API
// to pass variables between different domains and programming languaes.
//
// Originally based of https://pkg.go.dev/github.com/mkungla/vars/v5
package vars

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	EmptyVariable = Variable{}
	EmptyValue    = Value{}
	ErrValue      = errors.New("value error")
	ErrValueConv  = errors.New("failed to convert value")
	ErrKey        = errors.New("variable key error")
	ErrKeyEmpty   = fmt.Errorf("%w: key is empty", ErrKey)
)

func NewVariable(key string, val any, ro bool) (Variable, error) {
	key = strings.Trim(key, " ")

	if len(key) == 0 {
		return EmptyVariable, ErrKeyEmpty
	}
	if strings.Contains(key, " ") {
		return EmptyVariable, fmt.Errorf("%w: key can not contain spaces in the middle", ErrKey)
	}

	v, err := NewValue(val)
	return Variable{
		key: key,
		ro:  ro,
		val: v,
	}, err
}

func NewValue(val any) (Value, error) {
	if vv, ok := val.(Value); ok {
		if vv.Type() == TypeInvalid {
			return EmptyValue, fmt.Errorf("%w: source Value was invalid", ErrValue)
		}
		return vv, nil
	}
	if vv, ok := val.(Variable); ok {
		if vv.Type() == TypeInvalid {
			return EmptyValue, fmt.Errorf("%w: source Variable was invalid", ErrValue)
		}
		return vv.Value(), nil
	}
	p := getParser()
	defer p.free()

	typ, err := p.parseValue(val)
	v := Value{
		raw:      p.val,
		typ:      typ,
		str:      string(p.buf),
		isCustom: p.isCustom,
	}
	return v, err
}

// ParseKeyValue parses variable from single "key=val" pair and returns Variable.
func ParseKeyValue(kv string) (Variable, error) {
	if len(kv) == 0 {
		return EmptyVariable, ErrKeyEmpty
	}
	key, val, _ := strings.Cut(kv, "=")

	key = strings.Trim(key, " ")
	key = strings.Trim(key, "\\\"")

	val = strings.Trim(val, " ")

	if len(val) > 0 && val[0] == '"' {
		val = val[1:]
		if i := len(val) - 1; i >= 0 && val[i] == '"' {
			val = val[:i]
		}
	}

	return NewVariable(key, val, false)
}

// ParseTypedVariable parses variable and returns parser error for given type
// if parsing to requested type fails.
func ParseTypedVariable(key, val string, ro bool, typ Type) (Variable, error) {
	v, err := ParseTypedValue(val, typ)
	if err != nil {
		return EmptyVariable, err
	}
	return NewVariable(key, v, ro)
}

func NewTypedVariable(key string, val any, ro bool, typ Type) (Variable, error) {
	v, err := NewTypedValue(val, typ)
	if err != nil {
		return EmptyVariable, err
	}
	return NewVariable(key, v, ro)
}

func NewTypedValue(val any, typ Type) (Value, error) {
	src, err := NewValue(val)
	if err != nil {
		return EmptyValue, err
	}
	if src.Type() == typ {
		return src, nil
	}
	return ParseTypedValue(src.String(), typ)
}

func ParseTypedValue(val string, typ Type) (Value, error) {
	if typ == TypeString {
		return Value{
			typ: TypeString,
			str: val,
			raw: val,
		}, nil
	}
	var str string
	var err error
	var raw any
	switch typ {
	case TypeBool:
		raw, str, err = parseBool(val)
	case TypeFloat32:
		var rawd float64
		rawd, str, err = parseFloat(val, 32)
		raw = float32(rawd)
	case TypeFloat64:
		raw, str, err = parseFloat(val, 64)
	case TypeComplex64:
		raw, str, err = parseComplex64(val)
	case TypeComplex128:
		raw, str, err = parseComplex128(val)
	case TypeInt, TypeInt8, TypeInt16, TypeInt32, TypeInt64:
		raw, str, err = parseInts(val, typ)
	case TypeUint, TypeUint8, TypeUint16, TypeUint32, TypeUint64:
		raw, str, err = parseUints(val, typ)
	case TypeUintptr:
		var rawd uint64
		rawd, str, err = parseUint(val, 10, 64)
		raw = uintptr(rawd)
	default:
		err = fmt.Errorf("%w: can not create typed value %v from %s", ErrValue, typ, val)
	}

	if err != nil {
		err = fmt.Errorf("%w: can not parse %s as %s", err, val, typ)
		typ = TypeInvalid
	}

	return Value{
		raw: raw,
		typ: typ,
		str: str,
	}, err
}

func ValueOf(val any) Value {
	v, _ := NewValue(val)
	return v
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

type VariableIface[V ValueIface] struct {
	ro  bool
	key string
	val ValueIface
}

func (gvar VariableIface[V]) Value() V {
	return gvar.val.(V)
}

func (gvar VariableIface[V]) Key() string {
	return gvar.key
}

func (gvar VariableIface[V]) ReadOnly() bool {
	return gvar.ro
}

func (gvar VariableIface[V]) String() string {
	return gvar.val.String()
}

// ValueIface is minimal interface for Value to implement
// by thirtparty libraries
type ValueIface interface {
	// String MUST return string value Value
	fmt.Stringer
	// Underlying MUST return original value from what this
	// Value was created.
	Underlying() any
}

func As[VAL ValueIface](in Variable) VariableIface[VAL] {
	return VariableIface[VAL]{
		ro:  in.ReadOnly(),
		key: in.Key(),
		val: in.Value(),
	}
}
