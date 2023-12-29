// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

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
		{"basic", []string{"--basic", "1.0"}, 1, 15, true, nil, nil},
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
				t.Errorf("failed to parse int flag expected %t,%q got %v, err(%#v) (%d)", tt.ok, tt.err, ok, err, flag.Value())
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
