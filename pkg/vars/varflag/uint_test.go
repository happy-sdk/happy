// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package varflag

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestUintFlag(t *testing.T) {
	var tests = []struct {
		name   string
		in     []string
		want   uint
		defval uint
		ok     bool
		err    error
		cerr   error
	}{
		{"basic", []string{"--basic", "1"}, 1, 10, true, nil, nil},
		{"", []string{"--basic", "1"}, 0, 10, false, nil, ErrFlag},
		{"basic", []string{"--basic", "0"}, 0, 11, true, nil, nil},
		{"basic", []string{"--basic", strconv.FormatUint(math.MaxUint64, 10)}, math.MaxUint64, 12, true, nil, nil},
		{"basic", []string{"--basic", strconv.FormatUint(math.MaxUint64, 10)}, math.MaxUint64, 13, true, nil, nil},
		{"basic", []string{"--basic", "1000"}, 1000, 14, true, nil, nil},
		{"basic", []string{"--basic", "1.0"}, 15, 15, true, ErrInvalidValue, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, err := Uint(tt.name, tt.defval, "")
			if !errors.Is(err, tt.cerr) {
				t.Errorf("expected err to be %#v got %#v", tt.cerr, err)
			}
			if err != nil {
				return
			}
			if ok, err := flag.Parse(tt.in); ok != tt.ok || !errors.Is(err, tt.err) {
				t.Errorf("failed to parse uint flag expected %t,%q got %t,%#v (%d)", tt.ok, tt.err, ok, err, flag.Value())
			}

			if flag.Value() != tt.want {
				t.Errorf("provided %q expected value to be %d got %d", tt.in, tt.want, flag.Value())
			}
			if tt.ok && uint64(tt.want) != flag.Var().Uint64() {
				t.Errorf("expected uint(%d) convert to uint64 %d got %d", flag.Value(), uint64(tt.want), flag.Var().Uint64())
			}

			flag.Unset()
			if tt.ok && flag.Value() != tt.defval {
				t.Errorf("expected value to be %d got %d", tt.defval, flag.Value())
			}

			if flag.Present() {
				t.Error("expected flag to be unset")
			}
		})
	}
}
