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
	"time"
)

var (
	// ErrVariableKeyEmpty is used when variable key is empty string.
	ErrVariableKeyEmpty = errors.New("variable key can not be empty")

	// EmptyVar variable.
	EmptyVariable = Variable{}
	EmptyValue    = Value{}
)

type (
	NewVariableFunc func() (Variable, error)
)

// New return untyped Variable, If error occurred while parsing
// Variable represents default 0, nil value.
func New(key string, val any) Variable {
	return Variable{
		key: key,
		val: NewValue(val),
	}
}

// String returns modifiable Variable with string value.
// ro argument marks it's Value to be read-only.
func String(k string, v string, ro bool) (Variable, error) {
	return Variable{
		ro:  ro,
		key: k,
		val: NewValue(v),
	}, nil
}

// StringFunc
func StringFunc(k string, v string, ro bool) NewVariableFunc {
	return func() (Variable, error) {
		return String(k, v, ro)
	}
}

// Bool returns modifiable Variable with bool value.
// ro argument marks it's Value to be read-only.
func Bool(k string, v bool, ro bool) (Variable, error) {
	return Variable{
		ro:  ro,
		key: k,
		val: NewValue(v),
	}, nil
}

// BoolFunc
func BoolFunc(k string, v bool, ro bool) NewVariableFunc {
	return func() (Variable, error) {
		return Bool(k, v, ro)
	}
}

// Uint returns modifiable Variable with uint value.
// ro argument marks it's Value to be read-only.
func Uint(k string, v uint, ro bool) (Variable, error) {
	return Variable{
		ro:  ro,
		key: k,
		val: NewValue(v),
	}, nil
}

// UintFunc
func UintFunc(k string, v uint, ro bool) NewVariableFunc {
	return func() (Variable, error) {
		return Uint(k, v, ro)
	}
}

// Duration returns modifiable Variable with time.Duration value.
// ro argument marks it's Value to be read-only.
func Duration(k string, v time.Duration, ro bool) (Variable, error) {
	return Variable{
		ro:  ro,
		key: k,
		val: NewValue(v),
	}, nil
}

// DurationFunc
func DurationFunc(k string, v time.Duration, ro bool) NewVariableFunc {
	return func() (Variable, error) {
		return Duration(k, v, ro)
	}
}

// Float64 returns modifiable Variable with float64 value.
// ro argument marks it's Value to be read-only.
func Float64(k string, v float64, ro bool) (Variable, error) {
	return Variable{
		ro:  ro,
		key: k,
		val: NewValue(v),
	}, nil
}

// Float64Func
func Float64Func(k string, v float64, ro bool) NewVariableFunc {
	return func() (Variable, error) {
		return Float64(k, v, ro)
	}
}

// Int returns modifiable Variable with int value.
// ro argument marks it's Value to be read-only.
func Int(k string, v int, ro bool) (Variable, error) {
	return Variable{
		ro:  ro,
		key: k,
		val: NewValue(v),
	}, nil
}

// IntFunc
func IntFunc(k string, v int, ro bool) NewVariableFunc {
	return func() (Variable, error) {
		return Int(k, v, ro)
	}
}

// NewValue return Value created from provided argument,
// If error occurred while parsing Value represents 0|nil|"" value.
func NewValue(val any) Value {
	if vv, ok := val.(Value); ok {
		return vv
	}
	p := getParser()
	t, raw := p.parseArg(val)
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
