// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package typesinternal

// src: https://cs.opensource.google/go/x/tools/+/master:internal/typesinternal/recv.go
import "go/types"

// ReceiverNamed returns the named type (if any) associated with the
// type of recv, which may be of the form N or *N, or aliases thereof.
// It also reports whether a Pointer was present.
//
// The named result may be nil if recv is from a method on an
// anonymous interface or struct types or in ill-typed code.
func ReceiverNamed(recv *types.Var) (isPtr bool, named *types.Named) {
	t := recv.Type()
	if ptr, ok := types.Unalias(t).(*types.Pointer); ok {
		isPtr = true
		t = ptr.Elem()
	}
	named, _ = types.Unalias(t).(*types.Named)
	return
}
