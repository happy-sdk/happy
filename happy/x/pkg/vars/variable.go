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

package vars

// Variable is universal representation of key val pair.
type Variable struct {
	ro  bool
	key string
	val Value
}

// Key returns assigned key for this variable.
func (v Variable) Key() string {
	return v.key
}

// ReadOnly returns true if variable value is readonly.
func (v Variable) ReadOnly() bool {
	return v.ro
}

// Value returns Value of variable.
func (v Variable) Value() Value {
	return v.val
}

// String returns string representation of the var value.
func (v Variable) String() string {
	return v.val.String()
}

// Type of value.
func (v Variable) Type() Type {
	return v.val.typ
}

func (v Variable) Underlying() any {
	return v.val.raw
}

// Empty returns true if this Value is empty.
func (v Variable) Empty() bool {
	return v.val.Empty()
}

// Len returns the length of the string representation of the Value.
func (v Variable) Len() int {
	return v.val.Len()
}

func (v Variable) Bool() bool {
	vv, _ := v.val.Bool()
	return vv
}

// Int returns int representation of the var value.
func (v Variable) Int() int {
	vv, _ := v.val.Int()
	return vv
}

// Int8 returns int8 representation of the Value.
func (v Variable) Int8() int8 {
	vv, _ := v.val.Int8()
	return vv
}

// Int16 returns int16 representation of the Value.
func (v Variable) Int16() int16 {
	vv, _ := v.val.Int16()
	return vv
}

// Int32 returns int32 representation of the Value.
func (v Variable) Int32() int32 {
	vv, _ := v.val.Int32()
	return vv
}

// Int64 returns int64 representation of the Value.
func (v Variable) Int64() int64 {
	vv, _ := v.val.Int64()
	return vv
}

// Uint returns uint representation of the Value
func (v Variable) Uint() uint {
	vv, _ := v.val.Uint()
	return vv
}

// Uint8 returns uint8 representation of the Value.
func (v Variable) Uint8() uint8 {
	vv, _ := v.val.Uint8()
	return vv
}

// Uint16 returns uint16 representation of the Value.
func (v Variable) Uint16() uint16 {
	vv, _ := v.val.Uint16()
	return vv
}

// Uint32 returns uint32 representation of the Value.
func (v Variable) Uint32() uint32 {
	vv, _ := v.val.Uint32()
	return vv
}

// Uint64 returns uint64 representation of the Value.
func (v Variable) Uint64() uint64 {
	vv, _ := v.val.Uint64()
	return vv
}

// Float32 returns Float32 representation of the Value.
func (v Variable) Float32() float32 {
	vv, _ := v.val.Float32()
	return vv
}

// Float64 returns float64 representation of Value.
func (v Variable) Float64() float64 {
	vv, _ := v.val.Float64()
	return vv
}

// Complex64 returns complex64 representation of the Value.
func (v Variable) Complex64() complex64 {
	vv, _ := v.val.Complex64()
	return vv
}

// Complex128 returns complex128 representation of the Value.
func (v Variable) Complex128() complex128 {
	vv, _ := v.val.Complex128()
	return vv
}

// Uintptr returns uintptr representation of the Value.
func (v Variable) Uintptr() uintptr {
	vv, _ := v.val.Uintptr()
	return vv
}

// Fields calls strings.Fields on Value string.
func (v Variable) Fields() []string {
	return v.val.Fields()
}
