// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type testflag struct {
	name     string
	aliases  []string
	defval   interface{}
	required bool
	hidden   bool
	short    bool
	valid    bool
}

func testflags() []testflag {
	return []testflag{
		// valid
		{"flag", nil, "def-val", false, false, false, true},
		{"fl", nil, "def-val", false, false, false, true},
		{"flag", nil, "def-val", false, false, false, true},
		{"flag", nil, "def-val", false, false, false, true},
		{"flag2", nil, nil, false, true, false, true},
		{"flag3", nil, nil, true, false, false, true},
		{"flag-sub-1", []string{"alias", "a", "b"}, "flag sub", false, false, false, true},
		{"f", nil, "def-val", false, false, true, true},
		{"flag2", nil, "def-val", false, false, false, true},
		// invalid
		{"2", nil, nil, false, false, false, false},
		{" flag", []string{"flag", "flag2"}, nil, false, false, false, false},
		{"flag ", nil, nil, false, false, false, false},
		{"2flag", nil, nil, false, false, false, false},
	}
}

func TestName(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			flag, err := New(tt.name, tt.aliases...)
			if tt.valid {
				if err != nil {
					t.Errorf("valid flag %q did not expect error got %q", tt.name, err)
				}
				if n := flag.Name(); n != tt.name {
					t.Errorf("flag name should be %q got %q", tt.name, n)
				}
				return
			}
			if err == nil {
				t.Errorf("invalid flag %q expected error got <nil>", tt.name)
			}
			if flag != nil {
				t.Errorf("invalid flag %q should be <nil> got %#v", tt.name, flag)
			}
		})
	}
}

func TestFlag(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			expected := "-" + tt.name
			if !tt.short {
				expected = "-" + expected
			}
			got := flag.Flag()

			if got != expected {
				t.Errorf(".Flag want = %q, got = %q", expected, got)
			}
		})
	}
}

func TestNoArgs(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)

			if ok, err := flag.Parse([]string{}); ok || !errors.Is(err, ErrParse) {
				t.Errorf("flag should fail to parse got %t %q", ok, err)
			}
			flag2, _ := New(tt.name)
			if ok, err := flag2.Parse(nil); ok || !errors.Is(err, ErrParse) {
				t.Errorf("flag should fail to parse got %t %q", ok, err)
			}
		})
	}
}

func TestUsage(t *testing.T) {
	desc := "this is flag description"
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			flag.Default(tt.defval)
			flag.Usage(desc)

			expexted := desc
			if tt.defval != nil {
				expexted += fmt.Sprintf(" default: %q", fmt.Sprint(tt.defval))
			}
			if flag.Usage() != expexted {
				t.Errorf("Usage() want %q got %q", expexted, flag.Usage())
			}
		})
	}
}

func TestAliases(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name, tt.aliases...)
			if len(tt.aliases) != len(flag.Aliases()) {
				t.Errorf(
					"flag %q expected (%d) aliases got (%d) aliases - %s",
					tt.name,
					len(tt.aliases),
					len(flag.Aliases()),
					strings.Join(flag.Aliases(), ","),
				)
			}

			if len(tt.aliases) > 0 && len(flag.AliasesString()) <= len(tt.aliases) {
				t.Errorf("unexpected alias string %q", flag.AliasesString())
			}
		})
	}
}

func TestAliasesString(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name, tt.aliases...)
			if len(tt.aliases) > 0 {
				str := flag.AliasesString()
				for _, a := range tt.aliases {
					if !strings.Contains(str, a) {
						t.Errorf(
							"flag %q expected alias str to contain %q got (%q)",
							tt.name,
							a,
							str,
						)
					}
				}
			} else if len(flag.AliasesString()) != 0 {
				t.Error("expected no aliases for flag")
			}
		})
	}
}

func TestIsHidden(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			if tt.hidden {
				flag.Hide()
			}
			if tt.hidden != flag.IsHidden() {
				t.Errorf("flag should be hidden (%t) got (%t)", tt.hidden, flag.IsHidden())
			}
		})
	}
}

func TestGlobal(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			if flag.IsGlobal() {
				t.Error("flag should not be global by default")
			}

			flag.Parse([]string{"--" + tt.name, "some-value"})
			if flag.IsGlobal() {
				t.Error("flag should not be global after parsing from generic string")
			}

			flag2, _ := New(tt.name)
			flag2.Parse([]string{"/bin/app", "--" + tt.name, "some-value"})
			if !flag2.IsGlobal() {
				t.Error("flag should be global after parsing from os args")
			}
		})
	}
}

func TestAliasParse(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid || len(tt.aliases) == 0 || len(tt.aliases[0]) > 1 {
				return
			}
			flag, _ := New(tt.name, tt.aliases...)
			args := []string{"-" + tt.aliases[0], "some value for alias"}
			if ok, err := flag.Parse(args); !ok || err != nil {
				t.Error("expected string flag to parse alias, ", ok, err)
			}
		})
	}
}

func TestPos(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			if flag.Pos() != 0 {
				t.Errorf("flag.Pos want 0 got %d", flag.Pos())
			}
			args := []string{"arg1", "arg2", "--" + tt.name, "some value for alias"}
			if ok, err := flag.Parse(args); !ok || err != nil {
				t.Error("expected flag to parse, ", ok, err)
			}
			if flag.Pos() != 3 {
				t.Errorf("flag.Pos want 3 got %d", flag.Pos())
			}
		})
	}
}

func TestStringFlagPosition(t *testing.T) {
	flag, _ := New("position-flag", "a")
	if ok, err := flag.Parse([]string{"--some-flag=value2", "--position-flag=value1"}); !ok || err != nil {
		t.Error("expected string flag to parse, ", ok, err)
	}
	if flag.String() != "value1" {
		t.Errorf("expected string value to be \"value1\" got %q", flag.String())
	}
	if flag.Pos() != 3 {
		t.Errorf("expected position 3  got %d", flag.Pos())
	}
	if aliases := flag.Aliases(); len(aliases) != 1 || aliases[0] != "a" {
		t.Error("expected alias to be \"a\" got ", aliases)
	}
}

func TestRequired(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			if tt.required {
				flag.Required()
			}
			if tt.required != flag.IsRequired() {
				t.Error("flag should not be global by default")
			}

			if tt.required {
				ok, err := flag.Parse([]string{"some", "random", "args"})
				if ok || !errors.Is(err, ErrMissingRequired) {
					t.Errorf("expected parser to fail on required flag %t, %q", ok, err)
				}
			}
		})
	}
}

func TestStringFlagEmpty(t *testing.T) {
	flag, _ := New("some-flag")
	if ok, err := flag.Parse([]string{"--some-flag"}); ok || err == nil {
		t.Error("expected string flag parser to return not ok, ", ok, err)
	}
	if flag.String() != "" {
		t.Error("expected num value to be \"\" got ", flag.String())
	}
}

func TestUnset(t *testing.T) {
	tval := "test value"
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			flag.Default(tt.defval)
			args := []string{flag.Flag(), tval}
			flag.Parse(args)

			if flag.String() != tval {
				t.Errorf("expected flag var.String() to eq %q got %q", tval, flag.String())
			}

			flag.Unset()

			exp := ""
			if tt.defval != nil {
				exp = fmt.Sprint(tt.defval)
			}

			if flag.String() != exp {
				t.Errorf("expected flag value to eq %q got %q", exp, flag.String())
			}
		})
	}
}

func TestStringFlagPositionSpaces(t *testing.T) {
	flag1, _ := New("some-flag1")
	flag2, _ := New("some-flag2")
	flag3, _ := New("some-flag3")
	flag4, _ := New("n")
	args := []string{"-n", "a", "--some-flag1", "value1", "random", "--some-flag3", "value3", "--some-flag2", "value2"}

	if ok, err := flag1.Parse(args); !ok || err != nil {
		t.Error("expected string flag parser to return ok, ", ok, err)
	}
	if _, err := flag1.Parse(args); !errors.Is(err, ErrFlagAlreadyParsed) {
		t.Error("expected to fail on second parse, ", err)
	}
	if ok, err := flag2.Parse(args); !ok || err != nil {
		t.Error("expected string flag parser to return ok, ", ok, err)
	}
	if ok, err := flag3.Parse(args); !ok || err != nil {
		t.Error("expected string flag parser to return ok, ", ok, err)
	}
	if ok, err := flag4.Parse(args); !ok || err != nil {
		t.Error("expected string flag parser to return ok, ", ok, err)
	}
	if flag1.String() != "value1" {
		t.Error("expected some-flag1 value to be \"value1\" got ", flag1.String())
	}
	if flag2.String() != "value2" {
		t.Error("expected some-flag2 value to be \"value2\" got ", flag2.String())
	}
	if flag3.Value() != "value3" {
		t.Error("expected some-flag3 value to be \"value3\" got ", flag3.Value())
	}
	if flag4.Value() != "a" {
		t.Error("expected n value to be \"a\" got ", flag4.Value())
	}
}

func TestInvalidArgs(t *testing.T) {
	flag1, _ := New("some-flag1")
	args := []string{"-n", "a", "--some-flag1", "-=value", "randome", "--some-flag3", "value3", "--some-flag2", "value2"}

	if ok, err := flag1.Parse(args); ok || !errors.Is(err, ErrParse) {
		t.Error("expected string flag parser to fail, got", ok, err)
	}
}

func TestVariable(t *testing.T) {
	tval := "test value"
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				return
			}
			flag, _ := New(tt.name)
			args := []string{flag.Flag(), tval}
			if flag.IsGlobal() {
				t.Error("flag should be global by default")
			}

			ok, err := flag.Parse(args)
			if !ok {
				t.Errorf("expexted flag %q to parse", tt.name)
			}
			if err != nil {
				t.Errorf("dif not expect error while parsing %q got %q", tt.name, err)
			}
			if flag.IsGlobal() {
				t.Error("flag should be global")
			}
			v := flag.Var()
			if v.Key() != tt.name {
				t.Errorf("expected flag var.Key() to eq %q got %q", tt.name, v.Key())
			}
			if v.String() != tval {
				t.Errorf("expected flag var.String() to eq %q got %q", tval, v.String())
			}
		})
	}
}
