// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"regexp"
	"strings"
)

const (
	// Types define builtin type of the var
	TypeUnknown = iota
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
	TypeString
	TypeBytes
	TypeReflectVal
)

// New return untyped Variable
func New(key string, val string) Variable {
	return Variable{
		key:   key,
		vtype: TypeString,
		str:   val,
		raw:   val,
	}
}

// NewTyped parses variable and sets appropriately parser error for given type
// if parsing to requested type fails
func NewTyped(key string, val string, vtype uint) (Variable, error) {
	v := Variable{
		key:   key,
		vtype: vtype,
	}
	var err error
	if v.vtype == TypeString {
		v.raw = val
		v.str = val
	} else {
		switch v.vtype {
		case TypeBool:
			v.raw, v.str, err = parseBool(val)
		case TypeFloat32:
			r, s, e := parseFloat(val, 32)
			v.raw, v.str, err = float32(r), s, e
		case TypeFloat64:
			v.raw, v.str, err = parseFloat(val, 64)
		case TypeComplex64:
			v.raw, v.str, err = parseComplex64(val)
		case TypeComplex128:
			v.raw, v.str, err = parseComplex128(val)
		case TypeInt:
			r, s, e := parseInt(val, 10, 0)
			v.raw, v.str, err = int(r), s, e
		case TypeInt8:
			r, s, e := parseInt(val, 10, 8)
			v.raw, v.str, err = int8(r), s, e
		case TypeInt16:
			r, s, e := parseInt(val, 10, 16)
			v.raw, v.str, err = int16(r), s, e
		case TypeInt32:
			r, s, e := parseInt(val, 10, 32)
			v.raw, v.str, err = int32(r), s, e
		case TypeInt64:
			r, s, e := parseInt(val, 10, 64)
			v.raw, v.str, err = int64(r), s, e
		case TypeUint:
			r, s, e := parseUint(val, 10, 0)
			v.raw, v.str, err = uint(r), s, e
		case TypeUint8:
			r, s, e := parseUint(val, 10, 8)
			v.raw, v.str, err = uint8(r), s, e
		case TypeUint16:
			r, s, e := parseUint(val, 10, 16)
			v.raw, v.str, err = uint16(r), s, e
		case TypeUint32:
			r, s, e := parseUint(val, 10, 32)
			v.raw, v.str, err = uint32(r), s, e
		case TypeUint64:
			r, s, e := parseUint(val, 10, 64)
			v.raw, v.str, err = uint64(r), s, e
		case TypeUintptr:
			r, s, e := parseUint(val, 10, 64)
			v.raw, v.str, err = uintptr(r), s, e
		}
	}
	return v, err
}

// Parse return Variable, If error occurred while parsing
// Variable represents default 0, nil value
func Parse(key string, val interface{}) (Variable, error) {
	p := parser()
	vtype := p.printArg(val)
	s := string(p.buf)
	p.free()
	v := Variable{
		raw:   val,
		key:   key,
		str:   s,
		vtype: vtype,
	}
	return v, nil
}

// NewValue return Value,
func NewValue(val string) Value {
	return Value(val)
}

// ParseValue return Value, If error occurred while parsing
// VAR represents default 0, nil value
func ParseValue(val interface{}) (Value, error) {
	p := parser()
	p.printArg(val)
	s := Value(p.buf)
	p.free()
	return s, nil
}

// NewTypedValue parses value
func NewTypedValue(val string, vtype uint) (Value, error) {
	var v string
	var err error
	if vtype == TypeString {
		return Value(val), err
	}
	switch vtype {
	case TypeBool:
		_, v, err = parseBool(val)
	case TypeFloat32:
		_, v, err = parseFloat(val, 32)
	case TypeFloat64:
		_, v, err = parseFloat(val, 64)
	case TypeComplex64:
		_, v, err = parseComplex64(val)
	case TypeComplex128:
		_, v, err = parseComplex128(val)
	case TypeInt:
		_, v, err = parseInt(val, 10, 0)
	case TypeInt8:
		_, v, err = parseInt(val, 10, 8)
	case TypeInt16:
		_, v, err = parseInt(val, 10, 16)
	case TypeInt32:
		_, v, err = parseInt(val, 10, 32)
	case TypeInt64:
		_, v, err = parseInt(val, 10, 64)
	case TypeUint:
		_, v, err = parseUint(val, 10, 0)
	case TypeUint8:
		_, v, err = parseUint(val, 10, 8)
	case TypeUint16:
		_, v, err = parseUint(val, 10, 16)
	case TypeUint32:
		_, v, err = parseUint(val, 10, 32)
	case TypeUint64:
		_, v, err = parseUint(val, 10, 64)
	case TypeUintptr:
		_, v, err = parseUint(val, 10, 64)
	}
	return Value(v), err
}

// NewCollection returns new Collection
func NewCollection() Collection {
	return make(Collection)
}

// ParseKeyVal parses variable from single "key=val" pair and
// returns (key string, val Value)
func ParseKeyVal(kv string) (key string, val Value) {
	if len(kv) == 0 {
		return
	}
	reg := regexp.MustCompile(`"([^"]*)"`)

	kv = reg.ReplaceAllString(kv, "${1}")
	l := len(kv)
	for i := 0; i < l; i++ {
		if kv[i] == '=' {
			key = kv[:i]
			val = NewValue(kv[i+1:])
			if i < l {
				return
			}
		}
	}
	// VAR did not have any value
	key = kv[:l]
	val = ""
	return
}

// ParseKeyValSlice parses variables from any []"key=val" slice and
// returns Collection
func ParseKeyValSlice(kv []string) Collection {
	vars := NewCollection()
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
				vars[v[:i]] = NewValue(v[i+1:])
				if i < l {
					continue NextVar
				}
			}
		}
		// VAR did not have any value
		vars[strings.TrimRight(v[:l], "=")] = ""
	}
	return vars
}

// ParseFromBytes parses []bytes to string, creates []string by new line
// and calls ParseFromStrings.
func ParseFromBytes(b []byte) Collection {
	slice := strings.Split(string(b[0:]), "\n")
	return ParseKeyValSlice(slice)
}
