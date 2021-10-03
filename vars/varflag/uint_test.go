// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

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
		want   uint64
		defval uint64
		ok     bool
		err    error
	}{
		{"basic", []string{"--basic", "1"}, 1, 10, true, nil},
		{"basic", []string{"--basic", "0"}, 0, 11, true, nil},
		{"basic", []string{"--basic", strconv.FormatUint(math.MaxUint64, 10)}, math.MaxUint64, 12, true, nil},
		{"basic", []string{"--basic", strconv.FormatUint(math.MaxUint64, 10)}, math.MaxUint64, 13, true, nil},
		{"basic", []string{"--basic", "1000"}, 1000, 14, true, nil},
		{"basic", []string{"--basic", "1.0"}, 15, 15, true, ErrInvalidValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, _ := Uint(tt.name)
			flag.Default(tt.defval)
			if ok, err := flag.Parse(tt.in); ok != tt.ok || !errors.Is(err, tt.err) {
				t.Errorf("failed to parse uint flag expected %t,%q got %t,%#v (%d)", tt.ok, tt.err, ok, err, flag.Value())
			}

			if flag.Value() != tt.want {
				t.Errorf("expected value to be %d got %d", tt.want, flag.Value())
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
