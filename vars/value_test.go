// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueTypes(t *testing.T) {
	for _, test := range typeTests {
		v, _ := NewValue(test.in)
		assert.Equal(t, v.Bool(), test.bool)
		assert.Equal(t, v.Float32(), test.float32)
		assert.Equal(t, v.Float64(), test.float64)
		assert.Equal(t, v.Complex64(), test.complex64)
		assert.Equal(t, v.Complex128(), test.complex128)
		assert.Equal(t, v.Int(), test.int)
		assert.Equal(t, v.Int8(), test.int8)
		assert.Equal(t, v.Int16(), test.int16)
		assert.Equal(t, v.Int32(), test.int32)
		assert.Equal(t, v.Int64(), test.int64)
		assert.Equal(t, v.Uint(), test.uint)
		assert.Equal(t, v.Uint8(), test.uint8)
		assert.Equal(t, v.Uint16(), test.uint16)
		assert.Equal(t, v.Uint32(), test.uint32)
		assert.Equal(t, v.Uint64(), test.uint64)
		assert.Equal(t, v.Uintptr(), test.uintptr)
		assert.Equal(t, v.String(), test.string)
		assert.Equal(t, v.Bytes(), test.bytes)
		assert.Equal(t, v.Runes(), test.runes)
	}
}

func TestValueLen(t *testing.T) {
	for _, test := range typeTests {
		v, err := NewValue(test.in)
		assert.Equal(t, err, nil)
		assert.Equal(t, len(v.String()), len(test.in))
		assert.Equal(t, v.Len(), len(test.in))
	}
}
