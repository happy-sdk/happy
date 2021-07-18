// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
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

	// ErrVariableKeyEmpty is used when variable key is empty string
	ErrVariableKeyEmpty = errors.New("variable key can not be empty")

	// EmptyVar variable
	EmptyVar = Variable{}

	// parserPool is cached parser
	parserPool = sync.Pool{
		New: func() interface{} { return new(parser) },
	}

	// expunged is an arbitrary pointer that marks entries which have been deleted
	// from the dirty map.
	expunged = unsafe.Pointer(new(interface{}))
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

	// Collection is like a Go sync.Map safe for concurrent use
	// by multiple goroutines without additional locking or coordination.
	// Loads, stores, and deletes run in amortized constant time.
	//
	// The zero Map is empty and ready for use.
	// A Map must not be copied after first use.
	Collection struct {
		mu sync.Mutex

		// read contains the portion of the map's contents that are safe for
		// concurrent access (with or without mu held).
		//
		// The read field itself is always safe to load, but must only be stored with
		// mu held.
		//
		// Entries stored in read may be updated concurrently without mu, but updating
		// a previously-expunged entry requires that the entry be copied to the dirty
		// map and unexpunged with mu held.
		read atomic.Value // readOnly

		// dirty contains the portion of the map's contents that require mu to be
		// held. To ensure that the dirty map can be promoted to the read map quickly,
		// it also includes all of the non-expunged entries in the read map.
		//
		// Expunged entries are not stored in the dirty map. An expunged entry in the
		// clean map must be unexpunged and added to the dirty map before a new value
		// can be stored to it.
		//
		// If the dirty map is nil, the next write to the map will initialize it by
		// making a shallow copy of the clean map, omitting stale entries.
		dirty map[string]*entry

		// misses counts the number of loads since the read map was last updated that
		// needed to lock mu to determine whether the key was present.
		//
		// Once enough misses have occurred to cover the cost of copying the dirty
		// map, the dirty map will be promoted to the read map (in the unamended
		// state) and the next store to the map will make a new dirty copy.
		misses int

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

	// readOnly is an immutable struct stored atomically in the Map.read field.
	readOnly struct {
		m       map[string]*entry
		amended bool // true if the dirty map contains some key not in m.
	}

	// An entry is a slot in the map corresponding to a particular key.
	entry struct {
		// p points to the Variable stored for the entry.
		//
		// If p == nil, the entry has been deleted and m.dirty == nil.
		//
		// If p == expunged, the entry has been deleted, m.dirty != nil, and the entry
		// is missing from m.dirty.
		//
		// Otherwise, the entry is valid and recorded in m.read.m[key] and, if m.dirty
		// != nil, in m.dirty[key].
		//
		// An entry can be deleted by atomic replacement with nil: when m.dirty is
		// next created, it will atomically replace nil with expunged and leave
		// m.dirty[key] unset.
		//
		// An entry's associated value can be updated by atomic replacement, provided
		// p != expunged. If p == expunged, an entry's associated value can be updated
		// only after first setting m.dirty[key] = e so that lookups using the dirty
		// map find the entry.
		p unsafe.Pointer // *interface{}
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

// ParseKeyValSlice parses variables from any []"key=val" slice and
// returns Collection
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
				varr, _ := New(v[:i], v[i+1:])
				vars.Store(varr)
				if i < l {
					continue NextVar
				}
			}
		}
		// VAR did not have any value
		varr, _ := New(strings.TrimRight(v[:l], "="), "")
		vars.Store(varr)
	}
	return vars
}

// ParseFromBytes parses []bytes to string, creates []string by new line
// and calls ParseFromStrings.
func ParseFromBytes(b []byte) *Collection {
	slice := strings.Split(string(b[0:]), "\n")
	return ParseKeyValSlice(slice)
}
