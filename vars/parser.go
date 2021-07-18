// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"fmt"
	"reflect"
)

// getParser allocates a new parser struct or grabs a cached one.
func getParser() *parser {
	p := parserPool.Get().(*parser)
	p.panicking = false
	p.erroring = false
	p.fmt.init(&p.buf)
	return p
}

// Flag reports whether the flag c, a character, has been set.
// Satisfy fmt.State
func (p *parser) Flag(b int) bool {
	switch b {
	case '-':
		return p.fmt.minus
	case '+':
		return p.fmt.plus || p.fmt.plusV
	case '#':
		return p.fmt.sharp || p.fmt.sharpV
	case ' ':
		return p.fmt.space
	case '0':
		return p.fmt.zero
	}
	return false
}

// Precision returns the value of the precision option and whether it has been set.
// Satisfy fmt.State
func (p *parser) Precision() (prec int, ok bool) { return p.fmt.prec, p.fmt.precPresent }

// Width returns the value of the width option and whether it has been set.
// Satisfy fmt.State
func (p *parser) Width() (wid int, ok bool) { return p.fmt.wid, p.fmt.widPresent }

// Write so we can call Fprintf on a parser (through State), for
// recursive use in custom verbs.
func (p *parser) Write(b []byte) (ret int, err error) {
	p.buf.write(b)
	return len(b), nil
}

// free saves used parser structs in parserPool;
// that avoids an allocation per invocation.
func (p *parser) free() {
	// Proper usage of a sync.Pool requires each entry to have approximately
	// the same memory cost. To obtain this property when the stored type
	// contains a variably-sized buffer, we add a hard limit on the maximum buffer
	// to place back in the pool.
	//
	// See https://golang.org/issue/23199
	if cap(p.buf) > 64<<10 {
		return
	}
	p.buf = p.buf[:0]
	p.arg = nil
	p.value = reflect.Value{}
	parserPool.Put(p)
}

func (p *parser) printArg(arg interface{}) (vtype Type) {
	p.arg = arg
	p.value = reflect.Value{}
	if arg == nil {
		p.fmt.padString(nilAngleString)
		return
	}
	// Some types can be done without reflection.
	switch f := arg.(type) {
	case bool:
		p.fmt.boolean(f)
		vtype = TypeBool
	case float32:
		p.fmt.float(float64(f), 32, 'g', -1)
		vtype = TypeFloat32
	case float64:
		p.fmt.float(f, 64, 'g', -1)
		vtype = TypeFloat64
	case complex64:
		p.fmt.complex(complex128(f), 64)
		vtype = TypeComplex64
	case complex128:
		p.fmt.complex(f, 128)
		vtype = TypeComplex128
	case int:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt
	case int8:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt8
	case int16:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt16
	case int32:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt32
	case int64:
		p.fmt.integer(uint64(f), 10, signed, 'v', sdigits)
		vtype = TypeInt64
	case uint:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint
	case uint8:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint8
	case uint16:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint16
	case uint32:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint32
	case uint64:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUint64
	case uintptr:
		p.fmt.integer(uint64(f), 10, unsigned, 'v', udigits)
		vtype = TypeUintptr
	case string:
		p.fmt.string(f)
		vtype = TypeString
	case []byte:
		p.fmt.bytes(f, "[]byte")
		vtype = TypeBytes
	case reflect.Value:
		// Handle extractable values with special methods
		vtype = TypeReflectVal
		// since printValue does not handle them at depth 0.
		if f.IsValid() && f.CanInterface() {
			p.arg = f.Interface()
			if p.handleMethods() {
				return vtype
			}
		}
		p.printValue(f, 0)
	default:
		// If the type is not simple, it might have methods.
		vtype = TypeUnknown
		if !p.handleMethods() {
			// Need to use reflection, since the type had no
			// interface methods that could be used for formatting.
			p.printValue(reflect.ValueOf(f), 0)
		}
	}
	return vtype
}

// printValue is similar to printArg but starts with a reflect value, not an interface{} value.
// It does not handle 'p' and 'T' verbs because these should have been already handled by printArg.
func (p *parser) printValue(value reflect.Value, depth int) {
	// Handle values with special methods if not already handled by printArg (depth == 0).
	if depth > 0 && value.IsValid() && value.CanInterface() {
		p.arg = value.Interface()
		if p.handleMethods() {
			return
		}
	}
	p.arg = nil
	p.value = value

	switch f := value; value.Kind() {
	case reflect.Invalid:
		if depth == 0 {
			p.buf.writeString(invReflectString)
		} else {
			p.buf.writeString(nilAngleString)
		}
	case reflect.Bool:
		p.fmt.boolean(f.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.fmt.integer(uint64(f.Int()), 10, signed, 'v', sdigits)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p.fmt.integer(uint64(f.Int()), 10, signed, 'v', sdigits)
	case reflect.Float32:
		p.fmt.float(f.Float(), 32, 'g', -1)
	case reflect.Float64:
		p.fmt.float(f.Float(), 64, 'g', -1)
	case reflect.Complex64:
		p.fmt.complex(f.Complex(), 64)
	case reflect.Complex128:
		p.fmt.complex(f.Complex(), 128)
	case reflect.String:
		p.fmt.string(f.String())
	case reflect.Map:
		if p.fmt.sharpV {
			p.buf.writeString(f.Type().String())
			if f.IsNil() {
				p.buf.writeString(nilParenString)
				return
			}
			p.buf.writeByte('{')
		} else {
			p.buf.writeString(mapString)
		}
		sorted := sortMap(f)
		for i, key := range sorted.Key {
			if i > 0 {
				if p.fmt.sharpV {
					p.buf.writeString(commaSpaceString)
				} else {
					p.buf.writeByte(' ')
				}
			}
			p.printValue(key, depth+1)
			p.buf.writeByte(':')
			p.printValue(sorted.Value[i], depth+1)
		}
		if p.fmt.sharpV {
			p.buf.writeByte('}')
		} else {
			p.buf.writeByte(']')
		}
	case reflect.Struct:
		if p.fmt.sharpV {
			p.buf.writeString(f.Type().String())
		}
		p.buf.writeByte('{')
		for i := 0; i < f.NumField(); i++ {
			if i > 0 {
				if p.fmt.sharpV {
					p.buf.writeString(commaSpaceString)
				} else {
					p.buf.writeByte(' ')
				}
			}
			if p.fmt.plusV || p.fmt.sharpV {
				if name := f.Type().Field(i).Name; name != "" {
					p.buf.writeString(name)
					p.buf.writeByte(':')
				}
			}
			p.printValue(getField(f, i), depth+1)
		}
		p.buf.writeByte('}')
	case reflect.Interface:
		value := f.Elem()
		if !value.IsValid() {
			if p.fmt.sharpV {
				p.buf.writeString(f.Type().String())
				p.buf.writeString(nilParenString)
			} else {
				p.buf.writeString(nilAngleString)
			}
		} else {
			p.printValue(value, depth+1)
		}
	case reflect.Array, reflect.Slice:
		if p.fmt.sharpV {
			p.buf.writeString(f.Type().String())
			if f.Kind() == reflect.Slice && f.IsNil() {
				p.buf.writeString(nilParenString)
				return
			}
			p.buf.writeByte('{')
			for i := 0; i < f.Len(); i++ {
				if i > 0 {
					p.buf.writeString(commaSpaceString)
				}
				p.printValue(f.Index(i), depth+1)
			}
			p.buf.writeByte('}')
		}
	case reflect.Ptr:
		// pointer to array or slice or struct? ok at top level
		// but not embedded (avoid loops)
		if depth == 0 && f.Pointer() != 0 {
			switch a := f.Elem(); a.Kind() {
			case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
				p.buf.writeByte('&')
				p.printValue(a, depth+1)
				return
			}
		}
		fallthrough
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		p.pointer(f)
	default:
		p.unknownType(f)
	}
}

func (p *parser) handleMethods() (handled bool) {
	if p.erroring {
		return
	}

	verb := 'v'

	// Is it a Formatter?
	if formatter, ok := p.arg.(fmt.Formatter); ok {
		handled = true
		defer p.catchPanic(p.arg, verb, "Format")
		formatter.Format(p, verb)
		return
	}

	// Is it an error or Stringer?
	// The duplication in the bodies is necessary:
	// setting handled and deferring catchPanic
	// must happen before calling the method.
	switch v := p.arg.(type) {
	case error:
		handled = true
		defer p.catchPanic(p.arg, verb, "Error")
		p.fmt.string(v.Error())
		return

	case fmt.Stringer:
		handled = true
		defer p.catchPanic(p.arg, verb, "String")
		p.fmt.string(v.String())
		return
	}
	return false
}

func (p *parser) pointer(value reflect.Value) {
	var u uintptr
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		u = value.Pointer()
	default:
		p.badVerb()
		return
	}

	if u == 0 {
		p.fmt.padString(nilAngleString)
	} else {
		p.fmt0x64(uint64(u), !p.fmt.sharp)
	}
}

func (p *parser) catchPanic(arg interface{}, verb rune, method string) {
	if err := recover(); err != nil {
		// If it's a nil pointer, just say "<nil>". The likeliest causes are a
		// Stringer that fails to guard against nil or a nil pointer for a
		// value receiver, and in either case, "<nil>" is a nice result.
		if v := reflect.ValueOf(arg); v.Kind() == reflect.Ptr && v.IsNil() {
			p.buf.writeString(nilAngleString)
			return
		}
		// Otherwise print a concise panic message. Most of the time the panic
		// value will print itself nicely.
		if p.panicking {
			// Nested panics; the recursion in printArg cannot succeed.
			panic(err)
		}

		oldFlags := p.fmt.parserFmtFlags
		// For this output we want default behavior.
		p.fmt.clearflags()

		p.buf.writeString(percentBangString)
		p.buf.writeRune(verb)
		p.buf.writeString(panicString)
		p.buf.writeString(method)
		p.buf.writeString(" method: ")
		p.panicking = true
		p.printArg(err)
		p.panicking = false
		p.buf.writeByte(')')

		p.fmt.parserFmtFlags = oldFlags
	}
}

func (p *parser) unknownType(v reflect.Value) {
	if !v.IsValid() {
		p.buf.writeString(nilAngleString)
		return
	}
	p.buf.writeByte('?')
	p.buf.writeString(v.Type().String())
	p.buf.writeByte('?')
}

func (p *parser) badVerb() {
	p.erroring = true
	p.buf.writeString(percentBangString)
	p.buf.writeRune('v')
	p.buf.writeByte('(')
	switch {
	case p.arg != nil:
		p.buf.writeString(reflect.TypeOf(p.arg).String())
		p.buf.writeByte('=')
		p.printArg(p.arg)
	case p.value.IsValid():
		p.buf.writeString(p.value.Type().String())
		p.buf.writeByte('=')
		p.printValue(p.value, 0)
	default:
		p.buf.writeString(nilAngleString)
	}
	p.buf.writeByte(')')
	p.erroring = false
}

// fmt0x64 formats a uint64 in hexadecimal and prefixes it with 0x or
// not, as requested, by temporarily setting the sharp flag.
func (p *parser) fmt0x64(v uint64, leading0x bool) {
	sharp := p.fmt.sharp
	p.fmt.sharp = leading0x
	p.fmt.integer(v, 16, unsigned, 'v', sdigits)
	p.fmt.sharp = sharp
}
