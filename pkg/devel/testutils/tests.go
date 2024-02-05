// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package testutils

type Data[IN, WANT any] []struct {
	Name string
	In   IN
	Want WANT
}

type Result[T comparable] struct {
	Title string
	Want  T
	Got   T
}

type Results[T comparable] []Result[T]

func NewResults[T comparable]() Results[T] {
	return Results[T]{}
}

func (td *Results[T]) Add(name string, want, got T) {
	*td = append(*td, Result[T]{name, want, got})
}

func (td Results[T]) Test(ti TestingIface) {
	for _, test := range td {
		equal(ti, test.Want, test.Got, test.Title)
	}
}
