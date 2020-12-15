// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func parseFmt(key string, val interface{}) Variable {
	var vtype uint
	switch val.(type) {
	case bool:
		vtype = TypeBool
	case float32:
		vtype = TypeFloat32
	case float64:
		vtype = TypeFloat64
	case complex64:
		vtype = TypeComplex64
	case complex128:
		vtype = TypeComplex128
	case int:
		vtype = TypeInt
	case int8:
		vtype = TypeInt8
	case int16:
		vtype = TypeInt16
	case int32:
		vtype = TypeInt32
	case int64:
		vtype = TypeInt64
	case uint:
		vtype = TypeUint
	case uint8:
		vtype = TypeUint8
	case uint16:
		vtype = TypeUint16
	case uint32:
		vtype = TypeUint32
	case uint64:
		vtype = TypeUint64
	case uintptr:
		vtype = TypeUintptr
	case string:
		vtype = TypeString
	case []byte:
		vtype = TypeBytes
	case reflect.Value:
		vtype = TypeReflectVal
	default:
		vtype = TypeUnknown
	}
	return Variable{
		key:   key,
		str:   fmt.Sprintf("%v", val),
		raw:   val,
		vtype: vtype,
	}
}

func randString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

// Benchmark
// go test -bench BenchmarkNew -benchmem
func BenchmarkNew(b *testing.B) {
	// cached var
	b.Run("string:repeat", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := New("bm", "benchmark")
			if v.String() != "benchmark" {
				b.Error("Unexpected result: " + v.String())
			}
		}
	})

	b.Run("string:unique", func(b *testing.B) {
		// fixed := 100000
		vals := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			vals[i] = randString(9)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val := vals[i]
			v := New("bm", val)
			if v.String() != val {
				b.Error("Unexpected result: " + v.String())
			}
		}
	})
}

// Benchmark
// go test -bench BenchmarkParse -benchmem
func BenchmarkParse(b *testing.B) {
	// cached var
	b.Run("string:cached:pkg", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, _ := Parse("bm", "benchmark")
			if v.String() != "benchmark" {
				b.Error("Unexpected result: " + v.String())
			}
		}
	})
	b.Run("string:cached:fmt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := parseFmt("bm", "benchmark")
			if v.String() != "benchmark" {
				b.Error("Unexpected result: " + v.String())
			}
		}
	})

	b.Run("string:pkg", func(b *testing.B) {
		// fixed := 100000
		vals := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			vals[i] = randString(9)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val := vals[i]
			v, _ := Parse("bm", val)
			if v.String() != val {
				b.Error("Unexpected result: " + v.String())
			}
		}
	})

	b.Run("string:fmt", func(b *testing.B) {
		// fixed := 100000
		vals := make([]string, b.N)
		for i := 0; i < b.N; i++ {
			vals[i] = randString(9)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val := vals[i]
			v := parseFmt("bm", val)
			if v.String() != val {
				b.Error("Unexpected result: " + v.String())
			}
		}
	})

	b.Run("int:pkg", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, _ := Parse("bm", b.N)
			if v.Int() != b.N {
				b.Errorf("Unexpected result: %d", v.Int())
			}
		}
	})
	// cached var with fmt package
	b.Run("int:fmt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := parseFmt("bm", b.N)
			if v.Int() != b.N {
				b.Errorf("Unexpected result: %d", v.Int())
			}
		}
	})
}

// BenchmarkBool
// go test -bench BenchmarkBool -benchmem
func BenchmarkBool(b *testing.B) {
	b.Run("strconv", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			bool, _ := strconv.ParseBool("true")
			if bool != true {
				b.Errorf("Unexpected result: %t", bool)
			}
		}
	})
	b.Run("vartiable:new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := New("key", "true")
			if v.Bool() != true {
				b.Errorf("Unexpected result: %t", v.Bool())
			}
		}
	})
	b.Run("vartiable:new:typed", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, _ := NewTyped("key", "true", TypeBool)
			if v.Bool() != true {
				b.Errorf("Unexpected result: %t", v.Bool())
			}
		}
	})
	b.Run("vartiable:parse", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, _ := Parse("key", true)
			if v.Bool() != true {
				b.Errorf("Unexpected result: %t", v.Bool())
			}
		}
	})
	b.Run("value:new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v := Value("true")
			if v.Bool() != true {
				b.Errorf("Unexpected result: %t", v.Bool())
			}
		}
	})
}
