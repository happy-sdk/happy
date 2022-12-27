// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars

// A Kind represents the specific kind of kinde that a Value represents.
// The zero Kind is not a valid kind.
type Kind uint

const (
	KindInvalid Kind = iota
	KindBool
	KindInt
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindUintptr
	KindFloat32
	KindFloat64
	KindComplex64
	KindComplex128
	KindArray
	KindChan
	KindFunc
	KindInterface
	KindMap
	KindPointer
	KindSlice
	KindString
	KindStruct
	KindUnsafePointer
)

func (k Kind) String() (str string) {
	if uint(k) < uint(len(kindNames)) {
		str = kindNames[uint(k)]
	}
	return
}

var kindNames = []string{
	KindInvalid:       "invalid",
	KindBool:          "bool",
	KindInt:           "int",
	KindInt8:          "int8",
	KindInt16:         "int16",
	KindInt32:         "int32",
	KindInt64:         "int64",
	KindUint:          "uint",
	KindUint8:         "uint8",
	KindUint16:        "uint16",
	KindUint32:        "uint32",
	KindUint64:        "uint64",
	KindUintptr:       "uintptr",
	KindFloat32:       "float32",
	KindFloat64:       "float64",
	KindComplex64:     "complex64",
	KindComplex128:    "complex128",
	KindArray:         "array",
	KindChan:          "chan",
	KindFunc:          "func",
	KindInterface:     "interface",
	KindMap:           "map",
	KindPointer:       "ptr",
	KindSlice:         "slice",
	KindString:        "string",
	KindStruct:        "struct",
	KindUnsafePointer: "unsafe.Pointer",
}
