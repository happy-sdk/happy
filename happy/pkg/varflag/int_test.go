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
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestIntFlag(t *testing.T) {
	var tests = []struct {
		name   string
		in     []string
		want   int
		defval int
		ok     bool
		cerr   error
		err    error
	}{
		{"basic", []string{"--basic", "1"}, 1, 10, true, nil, nil},
		{"", []string{"--basic", "1"}, 1, 0, false, ErrFlag, nil},
		{"basic", []string{"--basic", "0"}, 0, 11, true, nil, nil},
		{"basic", []string{"--basic", fmt.Sprint(math.MaxInt64)}, math.MaxInt64, 12, true, nil, nil},
		{"basic", []string{"--basic", fmt.Sprint(math.MaxInt64)}, math.MaxInt64, 13, true, nil, nil},
		{"basic", []string{"--basic", "1000"}, 1000, 14, true, nil, nil},
		{"basic", []string{"--basic", "1.0"}, 15, 15, true, nil, ErrInvalidValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, err := Int(tt.name, tt.defval, "")
			if !errors.Is(err, tt.cerr) {
				t.Errorf("expected err to be %#v got %#v", tt.cerr, err)
			}
			if err != nil {
				return
			}
			if ok, err := flag.Parse(tt.in); ok != tt.ok || !errors.Is(err, tt.err) {
				t.Errorf("failed to parse int flag expected %t,%q got %t,%#v (%d)", tt.ok, tt.err, ok, err, flag.Value())
			}

			if flag.Value() != tt.want {
				t.Errorf("expected value to be %d got %d", tt.want, flag.Value())
			}
			if tt.ok && int64(tt.want) != flag.Var().Int64() {
				t.Errorf("expected int(%d) convert to int64 %d got %d", flag.Value(), int64(tt.want), flag.Var().Int64())
			}

			flag.Unset()
			if flag.Value() != tt.defval {
				t.Errorf("expected value to be %d got %d", tt.defval, flag.Value())
			}

			if flag.Present() {
				t.Error("expected flag to be unset")
			}
		})
	}
}
