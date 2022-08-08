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
	"testing"

	"github.com/mkungla/happy/pkg/vars"
)

func TestBoolFlagPresent(t *testing.T) {
	flag, _ := Bool("some-bool-flag", false, "")
	if present, err := flag.Parse([]string{"--some-bool-flag"}); !present || err != nil {
		t.Error("expected bool flag parser to return present, nil got ", present, err)
	}

	if !flag.Present() {
		t.Error("expected bool flag to be present")
	}

	if flag.Value() != true {
		t.Error("expected bool value to be true got ", flag.Var().Bool())
	}

	if flag.String() != "true" {
		t.Error("expected bool value to be \"true\" got ", flag.Var().String())
	}

	flag.Unset()
	if flag.Present() {
		t.Error("expected bool flag not to be present")
	}
}

type booltest struct {
	name  string
	alias string
	arg   string
	str   string
	b     bool
}

func booltests() []booltest {
	return []booltest{
		{"some-true-flag", "t", "true", "true", true},
		{"some-true-flag-on", "t", "on", "true", true},
		{"some-true-flag-1", "t", "1", "true", true},
		{"some-false-flag", "f", "false", "false", false},
		{"some-false-flag-off", "f", "off", "false", false},
		{"some-false-flag-0", "f", "0", "false", false},
	}
}
func TestBoolFlagValues(t *testing.T) {
	for _, tt := range booltests() {
		t.Run(tt.name, func(t *testing.T) {
			flag, _ := Bool(tt.name, !tt.b, "", tt.alias)
			args := fmt.Sprintf("--%s=%s", tt.name, tt.arg)

			if present, err := flag.Parse([]string{args}); !present || err != nil {
				t.Error("expected bool flag parser to return present, nil got ", present, err)
			}

			if !flag.Present() {
				t.Error("expected bool flag to be present")
			}
			if flag.Value() != tt.b {
				t.Errorf("expected bool value to be %t got %t", tt.b, flag.Value())
			}
			if flag.String() != tt.str {
				t.Errorf("expected bool value to be %q got %q", tt.str, flag.String())
			}
			if flag.Var().Type() != vars.TypeBool {
				t.Errorf("expected bool value Type to be TypeBool got %v", flag.Var().Type())
			}
			flag.Unset()
			if flag.Present() {
				t.Error("expected bool flag not to be present")
			}
			if flag.Value() != !tt.b {
				t.Errorf("expected %t got %t after unset", !tt.b, flag.Value())
			}
		})
	}
}

func TestBoolFlagValuesWithoutEq(t *testing.T) {
	for _, tt := range booltests() {
		t.Run(tt.name, func(t *testing.T) {
			flag, _ := Bool(tt.name, !tt.b, "", tt.alias)
			fname := fmt.Sprintf("--%s", tt.name)

			if present, err := flag.Parse([]string{fname, tt.arg}); !present || err != nil {
				t.Error("expected bool flag parser to return present, nil got ", present, err)
			}

			if !flag.Present() {
				t.Error("expected bool flag to be present")
			}
			if flag.Value() != tt.b {
				t.Errorf("expected bool value to be %t got %t", tt.b, flag.Value())
			}
			if flag.String() != tt.str {
				t.Errorf("expected bool value to be %q got %q", tt.str, flag.String())
			}
			if flag.Var().Type() != vars.TypeBool {
				t.Errorf("expected bool value Type to be TypeBool got %v", flag.Var().Type())
			}
			flag.Unset()
			if flag.Present() {
				t.Error("expected bool flag not to be present")
			}
			if flag.Value() != !tt.b {
				t.Errorf("expected %t got %t after unset", !tt.b, flag.Value())
			}
		})
	}
}

func TestBoolFlagAliasValues(t *testing.T) {
	for _, tt := range booltests() {
		t.Run(tt.name, func(t *testing.T) {
			flag, _ := Bool(tt.name, false, "", tt.alias)
			args := fmt.Sprintf("-%s=%s", tt.alias, tt.str)
			if present, err := flag.Parse([]string{args}); !present || err != nil {
				t.Error("expected bool flag parser to return present, nil got ", present, err)
			}
			if !flag.Present() {
				t.Error("expected bool flag to be present")
			}
			if flag.Value() != tt.b {
				t.Errorf("expected bool value to be %t got %t", tt.b, flag.Value())
			}
			if flag.String() != tt.str {
				t.Errorf("expected bool value to be %q got %q", tt.str, flag.String())
			}
		})
	}
}

func TestBoolFlagNotPresent(t *testing.T) {
	flag, _ := Bool("some-flag", false, "")
	if ok, err := flag.Parse([]string{"--some-flag-2"}); ok {
		t.Error("expected bool flag parser to return not ok, ", ok, err)
	}

	if flag.Present() {
		t.Error("expected bool flag not to be present")
	}

	if flag.Value() != false {
		t.Error("expected bool value to be false got ", flag.Value())
	}

	if flag.String() != "false" {
		t.Error("expected bool value to be \"false\" got ", flag.String())
	}
}

func TestBooltName(t *testing.T) {
	for _, tt := range testflags() {
		t.Run(tt.name, func(t *testing.T) {
			flag, err := Bool(tt.name, false, "")
			if !tt.valid {
				if err == nil {
					t.Errorf("invalid flag %q expected error got <nil>", tt.name)
				}
				if flag != nil {
					t.Errorf("invalid flag %q should be <nil> got %#v", tt.name, flag)
				}
			}
		})
	}
}
