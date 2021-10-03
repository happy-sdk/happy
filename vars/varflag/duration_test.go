// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"errors"
	"testing"
	"time"
)

func TestDurationFlag(t *testing.T) {
	var tests = []struct {
		name   string
		in     []string
		want   time.Duration
		defval time.Duration
		ok     bool
		err    error
	}{
		{"duration", []string{"--duration", "10s"}, 10000000000, 0, true, nil},
		{"duration", []string{"--duration", "1h10s"}, 3610000000000, 0, true, nil},
		{"duration", []string{"--duration", "1h30s"}, 3630000000000, 10, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, _ := Duration(tt.name)
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
