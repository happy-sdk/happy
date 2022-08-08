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
	TypeDuration
	TypeArray
)

type (
	// Type represents type of raw value.
	Type uint
)

//nolint: funlen, cyclop
func (t Type) String() string {
	switch t {
	case TypeUnknown:
		return "unknown"
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	case TypeFloat32:
		return "float32"
	case TypeFloat64:
		return "float64"
	case TypeComplex64:
		return "complex64"
	case TypeComplex128:
		return "complex128"
	case TypeInt:
		return "int"
	case TypeInt8:
		return "int8"
	case TypeInt16:
		return "int16"
	case TypeInt32:
		return "int32"
	case TypeInt64:
		return "int64"
	case TypeUint:
		return "uint"
	case TypeUint8:
		return "uint8"
	case TypeUint16:
		return "uint16"
	case TypeUint32:
		return "uint32"
	case TypeUint64:
		return "uint64"
	case TypeUintptr:
		return "uint64"
	case TypeBytes:
		return "bytes"
	case TypeRunes:
		return "runes"
	case TypeMap:
		return "map"
	case TypeReflectVal:
		return "reflect"
	case TypeDuration:
		return "duration"
	case TypeArray:
		return "array"
	}
	return ""
}
