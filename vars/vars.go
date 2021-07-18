// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
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

	signed   = true
	unsigned = false

	sdigits = "0123456789abcdefx"
	udigits = "0123456789ABCDEFX"

	nilAngleString    = "<nil>"
	percentBangString = "%!"
	panicString       = "(PANIC="
	invReflectString  = "<invalid reflect.Value>"
	nilParenString    = "(nil)"
	mapString         = "map["
	commaSpaceString  = ", "
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
		// value is used instead of arg for reflect values.
		value reflect.Value
		// panicking is set by catchPanic to avoid infinite panic,
		// recover, panic, ... recursion.
		panicking bool
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
		buf  *parserBuffer // buffer
		wid  int           // width
		prec int           // precision
		// intbuf is large enough to store %b of an int64 with a sign and
		// avoids padding at the end of the struct on 32 bit architectures.
		intbuf [68]byte
	}

	// parseBuffer is simple []byte instead of bytes.Buffer to avoid large dependency.
	parserBuffer []byte

	// parser fmt flags placed in a separate struct for easy clearing.
	parserFmtFlags struct {
		widPresent  bool
		precPresent bool
		minus       bool
		plus        bool
		sharp       bool
		space       bool
		zero        bool

		// For the formats %+v %#v, we set the plusV/sharpV flags
		// and clear the plus/sharp flags since %+v and %#v are in effect
		// different, flagless formats set at the top level.
		plusV  bool
		sharpV bool
	}

	// SortedMap represents a map's keys and values. The keys and values are
	// aligned in index order: Value[i] is the value in the map corresponding to Key[i].
	sortedMap struct {
		Key   []reflect.Value
		Value []reflect.Value
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
	t := p.printArg(val)
	s := Value{
		vtype: t,
		raw:   val,
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
	case TypeFloat64:
		raw, v, err = parseFloat(val, 64)
	case TypeComplex64:
		raw, v, err = parseComplex64(val)
	case TypeComplex128:
		raw, v, err = parseComplex128(val)
	case TypeInt:
		raw, v, err = parseInt(val, 10, 0)
	case TypeInt8:
		raw, v, err = parseInt(val, 10, 8)
	case TypeInt16:
		raw, v, err = parseInt(val, 10, 16)
	case TypeInt32:
		raw, v, err = parseInt(val, 10, 32)
	case TypeInt64:
		raw, v, err = parseInt(val, 10, 64)
	case TypeUint:
		raw, v, err = parseUint(val, 10, 0)
	case TypeUint8:
		raw, v, err = parseUint(val, 10, 8)
	case TypeUint16:
		raw, v, err = parseUint(val, 10, 16)
	case TypeUint32:
		raw, v, err = parseUint(val, 10, 32)
	case TypeUint64:
		raw, v, err = parseUint(val, 10, 64)
	case TypeUintptr:
		raw, v, err = parseUint(val, 10, 64)
	}
	return Value{
		raw:   raw,
		vtype: vtype,
		str:   v,
	}, err
}

// getField gets the i'th field of the struct value.
// If the field is itself is an interface, return a value for
// the thing inside the interface, not the interface itself.
func getField(v reflect.Value, i int) reflect.Value {
	val := v.Field(i)
	if val.Kind() == reflect.Interface && !val.IsNil() {
		val = val.Elem()
	}
	return val
}

// compare compares two values of the same type. It returns -1, 0, 1
// according to whether a > b (1), a == b (0), or a < b (-1).
// If the types differ, it returns -1.
// See the comment on Sort for the comparison rules.
func compare(aVal, bVal reflect.Value) int {
	aType, bType := aVal.Type(), bVal.Type()
	if aType != bType {
		return -1 // No good answer possible, but don't return 0: they're not equal.
	}
	switch aVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		a, b := aVal.Int(), bVal.Int()
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		a, b := aVal.Uint(), bVal.Uint()
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case reflect.String:
		a, b := aVal.String(), bVal.String()
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case reflect.Float32, reflect.Float64:
		return floatCompare(aVal.Float(), bVal.Float())
	case reflect.Complex64, reflect.Complex128:
		a, b := aVal.Complex(), bVal.Complex()
		if c := floatCompare(real(a), real(b)); c != 0 {
			return c
		}
		return floatCompare(imag(a), imag(b))
	case reflect.Bool:
		a, b := aVal.Bool(), bVal.Bool()
		switch {
		case a == b:
			return 0
		case a:
			return 1
		default:
			return -1
		}
	case reflect.Ptr:
		a, b := aVal.Pointer(), bVal.Pointer()
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case reflect.Chan:
		if c, ok := nilCompare(aVal, bVal); ok {
			return c
		}
		ap, bp := aVal.Pointer(), bVal.Pointer()
		switch {
		case ap < bp:
			return -1
		case ap > bp:
			return 1
		default:
			return 0
		}
	case reflect.Struct:
		for i := 0; i < aVal.NumField(); i++ {
			if c := compare(aVal.Field(i), bVal.Field(i)); c != 0 {
				return c
			}
		}
		return 0
	case reflect.Array:
		for i := 0; i < aVal.Len(); i++ {
			if c := compare(aVal.Index(i), bVal.Index(i)); c != 0 {
				return c
			}
		}
		return 0
	case reflect.Interface:
		if c, ok := nilCompare(aVal, bVal); ok {
			return c
		}
		c := compare(reflect.ValueOf(aVal.Elem().Type()), reflect.ValueOf(bVal.Elem().Type()))
		if c != 0 {
			return c
		}
		return compare(aVal.Elem(), bVal.Elem())
	default:
		// Certain types cannot appear as keys (maps, funcs, slices), but be explicit.
		panic("bad type in compare: " + aType.String())
	}
}

// nilCompare checks whether either value is nil. If not, the boolean is false.
// If either value is nil, the boolean is true and the integer is the comparison
// value. The comparison is defined to be 0 if both are nil, otherwise the one
// nil value compares low. Both arguments must represent a chan, func,
// interface, map, pointer, or slice.
func nilCompare(aVal, bVal reflect.Value) (int, bool) {
	if aVal.IsNil() {
		if bVal.IsNil() {
			return 0, true
		}
		return -1, true
	}
	if bVal.IsNil() {
		return 1, true
	}
	return 0, false
}

// floatCompare compares two floating-point values. NaNs compare low.
func floatCompare(a, b float64) int {
	switch {
	case isNaN(a):
		return -1 // No good answer if b is a NaN so don't bother checking.
	case isNaN(b):
		return 1
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func isNaN(a float64) bool {
	return a != a
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
	// s = strconv.FormatFloat(r, 'f', -1, bitSize)
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
	r = complex64(complex(f1, f2))
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
	f1 = float64(lf1)

	rf2, s2, err := parseFloat(fields[1], 64)
	if err != nil {
		return complex128(0), "", err
	}
	f2 = float64(rf2)
	s = s1 + " " + s2
	r = complex128(complex(f1, f2))
	return r, s, e
}

func parseInt(str string, base, bitSize int) (r int64, s string, e error) {
	r, e = strconv.ParseInt(str, base, bitSize)
	s = strconv.Itoa(int(r))
	return r, s, e
}

func parseUint(str string, base, bitSize int) (r uint64, s string, e error) {
	r, e = strconv.ParseUint(str, base, bitSize)
	s = strconv.Itoa(int(r))
	return r, s, e
}
