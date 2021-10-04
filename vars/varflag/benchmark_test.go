// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package varflag

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"
)

func BenchmarkString(b *testing.B) {
	args := []string{"/bin/app", "--str", "some value"}

	b.Run("pkg:string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			str, _ := New("string", "default str", "some string", "s", "str")
			str.Parse(args)
			str.Value()
		}
	})
}

func BenchmarkDuration(b *testing.B) {
	args := []string{"/bin/app", "--duration", "1h30s"}

	b.Run("pkg:duration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dur, _ := Duration("duration", 1*time.Second, "")
			dur.Parse(args)
			dur.Value()
		}
	})
}

func BenchmarkFloat(b *testing.B) {
	args := []string{"/bin/app", "--float", "1.001000023"}

	b.Run("pkg:float", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Float64("float", 1.0, "")
			f.Parse(args)
			f.Value()
		}
	})
}

func BenchmarkInt(b *testing.B) {
	args := []string{"/bin/app", "--int", fmt.Sprint(math.MinInt64), "int64"}
	b.Run("pkg:int", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Int("int", 1, "")
			f.Parse(args)
			f.Value()
		}
	})
}
func BenchmarkUint(b *testing.B) {
	args := []string{"/bin/app", "--uint", strconv.FormatUint(math.MaxUint64, 10), "uint64"}
	b.Run("pkg:uint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Uint("uint", 1, "")
			f.Parse(args)
			f.Value()
		}
	})
}

func BenchmarkBool(b *testing.B) {
	args := []string{"/bin/app", "--bool"}
	b.Run("pkg:uint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Bool("bool", false, "")
			f.Parse(args)
			f.Value()
		}
	})
}

func BenchmarkOption(b *testing.B) {
	args := []string{"/bin/app", "--option", "opt1", "--option", "opt2"}
	b.Run("pkg:uint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Option("option", []string{"defaultOpt"}, "", []string{"opt1", "opt2", "opt3", "defaultOpt"})
			f.Parse(args)
			f.Value()
		}
	})
}
