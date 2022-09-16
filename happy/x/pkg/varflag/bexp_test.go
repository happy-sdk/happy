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
	"testing"
)

func TestBexpFlag(t *testing.T) {
	flag, _ := Bexp("some-flag", "file-{a,b,c}.jpg", "expand path", "")
	if ok, err := flag.Parse([]string{"--some-flag", "file{0..2}.jpg"}); !ok || err != nil {
		t.Error("expected option flag parser to return ok, ", ok, err)
	}

	if flag.String() != "file0.jpg|file1.jpg|file2.jpg" {
		t.Error("expected option value to be \"file0.jpg|file1.jpg|file2.jpg\" got ", flag.String())
	}

	if len(flag.Value()) != 3 {
		t.Error("expected option value len to be \"3\" got ", len(flag.Value()))
	}

	if flag.Default().String() != "file-{a,b,c}.jpg" {
		t.Errorf("expected default to be file-{a,b,c}.jpg got %q", flag.Default().String())
	}
	if flag.String() != "file0.jpg|file1.jpg|file2.jpg" {
		t.Errorf("expected option value to be \"file0.jpg|file1.jpg|file2.jpg\" got %q", flag.String())
	}
	flag.Unset()

	if flag.String() != "file-{a,b,c}.jpg" {
		t.Errorf("expected option value to be \"file-a.jpg|file-b.jpg|file-c.jpg\" got %q", flag.String())
	}
}

func TestBexpFlagDefaults(t *testing.T) {
	flag, _ := Bexp("some-flag", "file-{a,b,c}.jpg", "expand path", "")
	if ok, err := flag.Parse([]string{""}); ok || err != nil {
		t.Error("expected option flag parser to return ok, ", ok, err)
	}

	if flag.String() != "file-{a,b,c}.jpg" {
		t.Errorf("expected option value to be \"file-{a,b,c}.jpg\" got %q", flag.String())
	}

	flag2, _ := Bexp("some-flag", "file-{x,y,z}.jpg", "expand path", "")
	if ok, err := flag2.Parse([]string{"some-flag", ""}); ok || err != nil {
		t.Error("expected option flag parser to return ok, ", ok, err)
	}

	if flag2.String() != "file-{x,y,z}.jpg" {
		t.Errorf("expected option value to be \"file-a.jpg|file-b.jpg|file-c.jpg\" got %q", flag2.String())
	}
}

func TestBexpInvalidName(t *testing.T) {
	_, err := Bexp("", "file-{a,b,c}.jpg", "expand path", "")
	if !errors.Is(err, ErrFlag) {
		t.Error("expected bexp flag parser to return err, ", err)
	}
}
