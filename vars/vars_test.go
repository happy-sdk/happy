// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"testing"
)

func TestParseFromString(t *testing.T) {
	key, val := ParseKeyVal("X=1")
	if key != "X" {
		t.Errorf("Key should be X got %q", key)
	}
	if val.Empty() {
		t.Error("Val should be 1")
	}
	if i := val.Int(); i != 1 {
		t.Error("ParseInt should be 1")
	}
}

func TestParseFromEmpty(t *testing.T) {
	if ek, ev := ParseKeyVal(""); ek != "" || ev != "" {
		t.Errorf("TestParseKeyValEmpty(\"\") = %q=%q, want ", ek, ev)
	}
	key, val := ParseKeyVal("X")
	if key != "X" {
		t.Errorf("Key should be X got %q", key)
	}
	if !val.Empty() {
		t.Error("Val should be empty")
	}
}
