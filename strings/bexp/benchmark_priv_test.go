// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2021 The Happy Authors

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

	re := regexp.MustCompile(`^-?0\d`)
	b.Run("GO/isPaddedRegexp", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, test := range tests {
				res := re.MatchString(test.in)
				if res != test.want {
					b.Errorf("expected %t got %t", !test.want, test.want)
				}
			}
		}
	})
}
