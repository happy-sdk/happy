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
			f, _ := New("string", "default str", "some string", "s", "str")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}

			f.Value()
		}
	})
}

func BenchmarkDuration(b *testing.B) {
	args := []string{"/bin/app", "--duration", "1h30s"}

	b.Run("pkg:duration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Duration("duration", 1*time.Second, "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}

func BenchmarkFloat(b *testing.B) {
	args := []string{"/bin/app", "--float", "1.001000023"}

	b.Run("pkg:float", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Float64("float", 1.0, "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}

func BenchmarkInt(b *testing.B) {
	args := []string{"/bin/app", "--int", fmt.Sprint(math.MinInt64), "int64"}
	b.Run("pkg:int", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Int("int", 1, "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}
func BenchmarkUint(b *testing.B) {
	args := []string{"/bin/app", "--uint", strconv.FormatUint(math.MaxUint64, 10), "uint64"}
	b.Run("pkg:uint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Uint("uint", 1, "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}

func BenchmarkBool(b *testing.B) {
	args := []string{"/bin/app", "--bool"}
	b.Run("pkg:uint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Bool("bool", false, "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}

func BenchmarkOption(b *testing.B) {
	args := []string{"/bin/app", "--option", "opt1", "--option", "opt2"}
	b.Run("pkg:uint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Option("option", []string{"defaultOpt"}, []string{"opt1", "opt2", "opt3", "defaultOpt"}, "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}

func BenchmarkBexp(b *testing.B) {
	args := []string{"/bin/app", "--some-flag", "file{0..2}.jpg"}
	b.Run("pkg:bexp", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f, _ := Bexp("some-flag", "file-{a,b,c}.jpg", "expand path", "")
			if _, err := f.Parse(args); err != nil {
				b.Error(err)
			}
			f.Value()
		}
	})
}
