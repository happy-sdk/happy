// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import (
	"regexp"
	"testing"
)

func BenchmarkIsPadded(b *testing.B) {
	tests := []struct {
		in   string
		want bool
	}{
		{"01", true},
		{"00001", true},
		{"-01", true},
		{"-00001", true},
		{"-00", true},
		{"00", true},
		{"00string", true},
		{"-00string", true},
	}
	b.Run("isPadded", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, test := range tests {
				res := isPadded(test.in)
				if res != test.want {
					b.Errorf("expected %t got %t", !test.want, test.want)
				}
			}
		}
	})
	b.Run("isPaddedRegexp", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, test := range tests {
				res := regexp.MustCompile(`^-?0\d`).Match([]byte(test.in))
				if res != test.want {
					b.Errorf("expected %t got %t", !test.want, test.want)
				}
			}
		}
	})
}
