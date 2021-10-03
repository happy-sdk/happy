// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestFloatFlag(t *testing.T) {
	var tests = []struct {
		name   string
		in     []string
		want   float64
		defval float64
		ok     bool
		err    error
	}{
		{"basic", []string{"--basic", "1"}, 1, 10, true, nil},
		{"basic", []string{"--basic", "0"}, 0, 11, true, nil},
		{"basic", []string{"--basic", fmt.Sprint(math.MaxFloat64)}, math.MaxFloat64, 12, true, nil},
		{"basic", []string{"--basic", fmt.Sprint(math.MaxFloat64)}, math.MaxFloat64, 13, true, nil},
		{"basic", []string{"--basic", "1000"}, 1000, 14, true, nil},
		{"basic", []string{"--basic", "1.0"}, 1.0, 15, true, nil},
		{"basic", []string{"--basic", "0.0001"}, 0.0001, 15, true, nil},
		{"basic", []string{"--basic", "-0.0001"}, -0.0001, 15, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, _ := Float(tt.name)
			flag.Default(tt.defval)
			if ok, err := flag.Parse(tt.in); ok != tt.ok || !errors.Is(err, tt.err) {
				t.Errorf("failed to parse uint flag expected %t,%q got %t,%#v (%f)", tt.ok, tt.err, ok, err, flag.Value())
			}

			if flag.Value() != tt.want {
				t.Errorf("expected value to be %f got %f", tt.want, flag.Value())
			}
			flag.Unset()
			if flag.Value() != tt.defval {
				t.Errorf("expected value to be %f got %f", tt.defval, flag.Value())
			}

			if flag.Present() {
				t.Error("expected flag to be unset")
			}
		})
	}
}
