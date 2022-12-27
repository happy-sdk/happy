// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package varflag

// "errors"
// "testing"
// "time"

// func TestDurationFlag(t *testing.T) {
// 	var tests = []struct {
// 		name   string
// 		in     []string
// 		want   time.Duration
// 		defval time.Duration
// 		ok     bool
// 		err    error
// 		cerr   error
// 	}{
// 		{"duration", []string{"--duration", "10s"}, 10000000000, 0, true, nil, nil},
// 		{"", []string{"--duration", "10s"}, 10000000000, 0, false, nil, ErrFlag},
// 		{"duration", []string{"--duration", "1h10s"}, 3610000000000, 0, true, nil, nil},
// 		{"duration", []string{"--duration", "1h30s"}, 3630000000000, 10, true, nil, nil},
// 		{"duration", []string{"--duration", "one hour"}, 10, 10, true, ErrInvalidValue, nil},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			flag, err := Duration(tt.name, tt.defval, "")
// 			if !errors.Is(err, tt.cerr) {
// 				t.Errorf("expected err to be %#v got %#v", tt.cerr, err)
// 			}
// 			if err != nil {
// 				return
// 			}
// 			if ok, err := flag.Parse(tt.in); ok != tt.ok || !errors.Is(err, tt.err) {
// 				t.Errorf("failed to parse duration flag expected %t,%q got %t,%#v (%d)", tt.ok, tt.err, ok, err, flag.Value())
// 			}

// 			if flag.Value() != tt.want {
// 				t.Errorf("expected value to be %d got %d", tt.want, flag.Value())
// 			}
// 			flag.Unset()
// 			if flag.Value() != tt.defval {
// 				t.Errorf("expected value to be %d got %d", tt.defval, flag.Value())
// 			}

// 			if flag.Present() {
// 				t.Error("expected flag to be unset")
// 			}
// 		})
// 	}
// }
