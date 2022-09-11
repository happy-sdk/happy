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
// formats/kinds to common key value pair. Or key value pair sets to Map.
//
// Main purpose of this library is to provide simple API
// to pass variables between different domains and programming languaes.
//
// Originally based of https://github.com/mkungla/vars
// and will be moved there as new module import path some point.

package vars

var (
	ErrValue     = newError("value error")
	ErrValueConv = wrapError(ErrValue, "failed to convert value")
	ErrKey       = newError("variable key error")

	EmptyVariable = Variable{}
	EmptyValue    = Value{}
)

func NewVariable(key string, val any, ro bool) (Variable, error) {
	key = stringsTrimSpace(key)

	if len(key) == 0 {
		return EmptyVariable, ErrKey
	}

	v, err := NewValue(val)
	return Variable{
		key: key,
		ro:  ro,
		val: v,
	}, err
}

func NewVariableAs(key string, val any, ro bool, kind Kind) (Variable, error) {
	v, err := NewValueAs(val, kind)
	if err != nil {
		return EmptyVariable, err
	}
	return NewVariable(key, v, ro)
}

func NewValue(val any) (Value, error) {
	if vv, ok := val.(Value); ok {
		if vv.Kind() == KindInvalid {
			return EmptyValue, wrapError(ErrValue, "source Value was invalid")
		}
		return vv, nil
	}
	if vv, ok := val.(Variable); ok {
		if vv.Kind() == KindInvalid {
			return EmptyValue, wrapError(ErrValue, "source Variable was invalid")
		}
		return vv.Value(), nil
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

// ParseKeyValue parses variable from single key=val pair and returns a Variable
// if parsing is successful. EmptyVariable and error is returned when parsing fails.
func ParseVariableFromString(kv string) (Variable, error) {
	if len(kv) == 0 {
		return EmptyVariable, ErrKey
	}
	k, v, _ := stringsFastCut(kv, '=')

	key, err := parseKey(k)
	if err != nil {
		return EmptyVariable, wrapError(err, "failed to parse variable key")
	}

	val := trimAndUnquoteValue(v)
	if err != nil {
		return EmptyVariable, wrapError(err, "failed to parse variable value")
	}

	return NewVariable(key, val, false)
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

// ParseKinddVariable parses variable and returns parser error for given kinde
// if parsing to requested kinde fails.
func ParseVariableAs(key, val string, ro bool, kind Kind) (Variable, error) {
	v, err := ParseValueAs(val, kind)
	if err != nil {
		return EmptyVariable, err
	}
	return NewVariable(key, v, ro)
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
		err = wrapError(ErrValue, "can not create kinded value "+kind.String()+" from "+val)
	}

	if err != nil {
		err = wrapError(err, "can not parse "+val+" as "+kind.String())
		kind = KindInvalid
	}

	return Value{
		raw:  raw,
		kind: kind,
		str:  str,
	}, err
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
		vars.Store(vv.Key(), vv.Value())
	}
	return vars, nil
}

// ParseFromBytes parses []bytes to string, creates []string by new line
// and calls ParseFromStrings.
func ParseMapFromBytes(b []byte) (*Map, error) {
	slice := stringsSplit(string(b[0:]), '\n')
	return ParseMapFromSlice(slice)
}

func ValueOf(val any) Value {
	v, _ := NewValue(val)
	return v
}

func ValueKindOf(in any) (kind Kind) {
	_, kind = underlyingValueOf(in, false)
	return
}

func ParseKey(str string) (key string, err error) {
	return parseKey(str)
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

func As[VAL ValueIface](in Variable) VariableIface[VAL] {
	return VariableIface[VAL]{
		ro:  in.ReadOnly(),
		key: in.Key(),
		val: in.Value(),
	}
}

// errorString is a trivial implementation of error.
type varsError struct {
	s string
}

func (e *varsError) Error() string {
	return e.s
}

// New returns an error that formats as the given text.
// Each call to New returns a distinct error value even if the text is identical.
func newError(text string) error {
	return &varsError{text}
}

func wrapError(err error, text string) error {
	return &wrappedError{
		err: err,
		msg: err.Error() + ": " + text,
	}
}

type wrappedError struct {
	msg string
	err error
}

func (e *wrappedError) Error() string {
	return e.msg
}

func (e *wrappedError) Unwrap() error {
	return e.err
}
