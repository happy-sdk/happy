// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.package flags

package varflag

import "testing"

func TestBexpFlag(t *testing.T) {
	flag, _ := Bexp("some-flag", "file-{a,b,c}.jpg", "expand path", "")
	if ok, err := flag.Parse([]string{"--some-flag", "file{0..2}.jpg"}); !ok || err != nil {
		t.Error("expected option flag parser to return ok, ", ok, err)
	}

	if flag.String() != "file0.jpg|file1.jpg|file2.jpg" {
		t.Error("expected option value to be \"file0.jpg|file1.jpg|file2.jpg\" got ", flag.String())
	}

	if flag.Default().String() != "file-{a,b,c}.jpg" {
		t.Errorf("expected default to be file-{a,b,c}.jpg got %q", flag.Default().String())
	}
	if flag.String() != "file0.jpg|file1.jpg|file2.jpg" {
		t.Errorf("expected option value to be \"file0.jpg|file1.jpg|file2.jpg\" got %q", flag.String())
	}
	flag.Unset()
	if flag.String() != "file-a.jpg|file-b.jpg|file-c.jpg" {
		t.Errorf("expected option value to be \"file-a.jpg|file-b.jpg|file-c.jpg\" got %q", flag.String())
	}
}

func TestBexpFlagDefaults(t *testing.T) {
	flag, _ := Bexp("some-flag", "file-{a,b,c}.jpg", "expand path", "")
	if ok, err := flag.Parse([]string{""}); ok || err != nil {
		t.Error("expected option flag parser to return ok, ", ok, err)
	}

	if flag.String() != "file-a.jpg|file-b.jpg|file-c.jpg" {
		t.Errorf("expected option value to be \"file-a.jpg|file-b.jpg|file-c.jpg\" got %q", flag.String())
	}
}
