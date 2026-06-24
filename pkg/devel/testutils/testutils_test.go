// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package testutils

import (
	"errors"
	"testing"
)

func TestShouldSucceed(t *testing.T) {
	var testErr = errors.New("test error")
	True(t, true, "ecpected true")
	False(t, false, "ecpected false")
	NoError(t, nil, "ecpected no error")
	ErrorIs(t, testErr, testErr, "ecpected error to be testErr")
	Equal(t, 1, 1)
	Equal(t, true, true)
	Equal(t, "nil", "nil")
	NotEqual(t, 1, 2, "1 should not equal 2")
	NotEqual(t, "a", "b", "a should not equal b")
}

// TestExtractCoverage is a regression test for ExtractCoverage: it had dead
// code where the generic field-splitting fallback always fired before the
// "[no test files]" check, and silently mislabeled a FAIL line's duration
// field as if it were a coverage percentage.
func TestExtractCoverage(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		want      string
		wantError bool
	}{
		{
			name: "pass with coverage",
			line: "ok  \tgithub.com/foo/bar\t0.123s\tcoverage: 85.3% of statements",
			want: "85.3%",
		},
		{
			name: "no test files",
			line: "?   \tgithub.com/foo/bar\t[no test files]",
			want: "no test files",
		},
		{
			name:      "failed, no coverage info",
			line:      "FAIL\tgithub.com/foo/bar\t0.123s",
			wantError: true,
		},
		{
			name:      "pass, no coverage info",
			line:      "ok  \tgithub.com/foo/bar\t0.123s",
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ExtractCoverage(test.line)
			if test.wantError {
				if err == nil {
					t.Errorf("expected error, got result %q", got)
				}
				if got != "" {
					t.Errorf("expected empty result on error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			Equal(t, test.want, got)
		})
	}
}

// fakeT is a minimal TestingIface that records failures instead of actually
// failing the enclosing test, so we can assert on the false/failing path of
// assertion helpers without the test runner itself reporting a failure.
type fakeT struct {
	failed bool
}

func (f *fakeT) Errorf(format string, args ...any) { f.failed = true }
func (f *fakeT) Name() string                      { return "fakeT" }
func (f *fakeT) Helper()                           {}

func TestLen(t *testing.T) {
	Assert(t, Len(t, "abc", 3), "expected Len to report true for matching length")
	Assert(t, Len(t, []int{1, 2}, 2), "expected Len to report true for matching slice length")
	Assert(t, Len(t, map[string]int{"a": 1}, 1), "expected Len to report true for matching map length")

	ft := &fakeT{}
	Assert(t, !Len(ft, "abc", 5), "expected Len to report false for mismatched length")
	Assert(t, ft.failed, "expected Len to report failure via Errorf for mismatched length")
}

func TestIsType(t *testing.T) {
	Assert(t, IsType(t, "", "hello"), "expected IsType to report true for matching types")
	Assert(t, IsType(t, 0, 1), "expected IsType to report true for matching int types")

	ft := &fakeT{}
	Assert(t, !IsType(ft, "", 1), "expected IsType to report false for mismatched types")
	Assert(t, ft.failed, "expected IsType to report failure via Errorf for mismatched types")
}

func TestNilAndNotNil(t *testing.T) {
	var p *int
	Assert(t, Nil(t, nil), "expected Nil to report true for untyped nil")
	Assert(t, Nil(t, p), "expected Nil to report true for typed nil pointer")

	v := 1
	Assert(t, NotNil(t, &v), "expected NotNil to report true for non-nil pointer")
	Assert(t, NotNil(t, "x"), "expected NotNil to report true for non-nil value")

	ft := &fakeT{}
	Assert(t, !Nil(ft, &v), "expected Nil to report false for non-nil value")
	Assert(t, ft.failed, "expected Nil to report failure via Errorf for non-nil value")

	ft2 := &fakeT{}
	Assert(t, !NotNil(ft2, nil), "expected NotNil to report false for nil value")
	Assert(t, ft2.failed, "expected NotNil to report failure via Errorf for nil value")
}

func TestHasPrefixAndHasSuffix(t *testing.T) {
	Assert(t, HasPrefix(t, "hello world", "hello"), "expected HasPrefix to report true")
	Assert(t, HasSuffix(t, "hello world", "world"), "expected HasSuffix to report true")

	ft := &fakeT{}
	Assert(t, !HasPrefix(ft, "hello world", "world"), "expected HasPrefix to report false for non-matching prefix")
	Assert(t, ft.failed, "expected HasPrefix to report failure via Errorf for non-matching prefix")

	ft2 := &fakeT{}
	Assert(t, !HasSuffix(ft2, "hello world", "hello"), "expected HasSuffix to report false for non-matching suffix")
	Assert(t, ft2.failed, "expected HasSuffix to report failure via Errorf for non-matching suffix")
}
