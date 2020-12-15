// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import "testing"

// Benchmark
// go test -bench BenchmarkNew -benchmem
func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := Parse("file-{a,b,c}.jpg")
		if v.String() != "benchmark" {
			b.Error("Unexpected result: " + v.String())
		}
	}
}
