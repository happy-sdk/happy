// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package testutils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// TestingIface is an interface wrapper for
// *testing.T, *testing.B, *testing.F
type TestingIface interface {
	Errorf(format string, args ...any)
	Name() string
	Helper()
}

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if testutils.NoError(t, err) {
//		   testutils.Equal(t, expectedObj, actualObj)
//	  }
func NoError(ti TestingIface, err error, msgAndArgs ...any) bool {
	if err != nil {
		ti.Helper()
		return fail(ti, fmt.Sprintf("received unexpected error:\n%+v", err), msgAndArgs...)
	}
	return true
}

// Error assert that err != nil
func Error(tt TestingIface, err error, msgAndArgs ...any) bool {
	if err == nil {
		tt.Helper()
		return fail(tt, "an error is expected but got nil.", msgAndArgs...)
	}

	return true
}

// ErrorIs asserts errors.Is(err, target)
func ErrorIs(tt TestingIface, err, target error, msgAndArgs ...any) bool {
	tt.Helper()
	if errors.Is(err, target) {
		return true
	}

	var expectedText string
	if target != nil {
		expectedText = target.Error()
	}

	chain := buildErrorChainString(err)

	return fail(tt, fmt.Sprintf("Target error should be in err chain:\n"+
		"expected: %q\n"+
		"got chain: %q", expectedText, chain,
	), msgAndArgs...)
}

// Equal asserts comparables.
//
//	testutils.Equal(t, "hello", "hello")
func Equal[V comparable](tt TestingIface, want, got V, msgAndArgs ...any) bool {
	tt.Helper()
	return equal(tt, want, got, msgAndArgs...)
}
func equal[V comparable](tt TestingIface, want, got V, msgAndArgs ...any) bool {
	tt.Helper()
	if got != want {
		return fail(tt, fmt.Sprintf("Not equal: \n"+
			"want: %v\n"+
			"got  : %v", want, got), msgAndArgs...)
	}
	return true
}

func HasPrefix(tt TestingIface, s, prefix string, msgAndArgs ...any) bool {
	tt.Helper()
	if !strings.HasPrefix(s, prefix) {
		return fail(tt, fmt.Sprintf("Does not have prefix: \n"+
			"expected: %v\n"+
			"actual  : %v", prefix, s), msgAndArgs...)
	}
	return true
}

func HasSuffix(tt TestingIface, s, suffix string, msgAndArgs ...any) bool {
	tt.Helper()
	if !strings.HasSuffix(s, suffix) {
		return fail(tt, fmt.Sprintf("Does not have suffix: \n"+
			"expected: %v\n"+
			"actual  : %v", suffix, s), msgAndArgs...)
	}
	return true
}

func NotEqual[V comparable](tt TestingIface, want, got V, msgAndArgs ...any) bool {
	tt.Helper()
	if got == got {
		return fail(tt, fmt.Sprintf("Equal: \n"+
			"expected: %v != %v", want, got), msgAndArgs...)
	}
	return true
}

func EqualAny(tt TestingIface, want, got any, msgAndArgs ...any) bool {
	tt.Helper()
	if err := validateEqualArgs(want, got); err != nil {
		return fail(tt, fmt.Sprintf("Invalid operation: %#v == %#v (%s)",
			want, got, err), msgAndArgs...)
	}

	if !ObjectsAreEqual(want, got) {
		want, got = formatUnequalValues(want, got)
		return fail(tt, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s", want, got), msgAndArgs...)
	}

	return true

}

func EqualAnyf(tt TestingIface, want, got any, msg string, args ...any) bool {
	tt.Helper()
	return EqualAny(tt, want, got, append([]any{msg}, args...)...)
}

// True asserts that the specified value is true.
//
//	testutils.True(t, myBool)
func True(tt TestingIface, value bool, msgAndArgs ...any) bool {
	if !value {
		tt.Helper()
		return fail(tt, "Should be true", msgAndArgs...)
	}
	return true
}

// False asserts that the specified value is false.
//
//	testutils.False(t, myBool)
func False(tt TestingIface, value bool, msgAndArgs ...any) bool {
	if value {
		tt.Helper()
		return fail(tt, "Should be false", msgAndArgs...)
	}
	return true
}

// NotNil asserts that the specified value is not nil.
//
//	testutils.NotNil(t, &val)
func NotNil(tt TestingIface, value any, msgAndArgs ...any) bool {
	if value == nil {
		tt.Helper()
		return fail(tt, "Should not be <nil>", msgAndArgs...)
	}
	return true
}

// Nil asserts that the specified value is nil.
func Nil(tt TestingIface, value any, msgAndArgs ...any) bool {
	if value != nil {
		tt.Helper()
		return fail(tt, "Should be <nil>", msgAndArgs...)
	}
	return true
}

type length interface {
	Len() int
}

// Len asserts that the specified value has the expected length.
func Len(tt TestingIface, value any, expected int, msgAndArgs ...any) bool {
	tt.Helper()

	if value == nil {
		return fail(tt, "Value should not be nil", msgAndArgs...)
	}

	var got int
	switch v := value.(type) {
	case length:
		got = v.Len()
	default:
		// Use reflection to check if value is a slice, map, string, or array
		val := reflect.ValueOf(v)
		switch val.Kind() {
		case reflect.Slice, reflect.Map, reflect.String, reflect.Array:
			got = val.Len()
		default:
			return fail(tt, fmt.Sprintf("Type %T does not support length checking", value), msgAndArgs...)
		}
	}

	if expected != got {
		return fail(tt, fmt.Sprintf("Expected length %d, got %d", expected, got), msgAndArgs...)
	}
	return true
}

func KeyValErrorMsg(key string, val any) string {
	return fmt.Sprintf("key(%v) = val(%v)", key, val)
}

// callerInfo returns an array of strings containing the file and line number
// of each stack frame leading from the current test to the assert call that
// failed.
//
// CallerInfo is necessary because the assert functions use the testing object
// internally, causing it to print the file:line of the assert method, rather than where
// the problem actually occurred in calling code.
func callerInfo() []string {

	var pc uintptr
	var ok bool
	var file string
	var line int
	var name string

	callers := []string{}
	for i := 0; ; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			// The breaks below failed to terminate the loop, and we ran off the
			// end of the call stack.
			break
		}

		// This is a huge edge case, but it will panic if this is the case, see #180
		if file == "<autogenerated>" {
			break
		}

		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		name = f.Name()

		// testing.tRunner is the standard library function that calls
		// tests. Subtests are called directly by tRunner, without going through
		// the Test/Benchmark/Example function that contains the t.Run calls, so
		// with subtests we should break when we hit tRunner, without adding it
		// to the list of callers.
		if name == "testing.tRunner" {
			break
		}

		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
		if len(parts) > 1 {
			dir := parts[len(parts)-2]
			if (dir != "assert" && dir != "mock" && dir != "require") || file == "mock_test.go" {
				path, _ := filepath.Abs(file)
				callers = append(callers, fmt.Sprintf("%s:%d", path, line))
			}
		}

		// Drop the package
		segments := strings.Split(name, ".")
		name = segments[len(segments)-1]
		if isTest(name, "Test") ||
			isTest(name, "Benchmark") ||
			isTest(name, "Example") {
			break
		}
	}

	return callers
}

func fail(tt TestingIface, failureMessage string, msgAndArgs ...any) bool {
	tt.Helper()
	content := []labeledContent{
		{"Error Trace", strings.Join(callerInfo(), "\n\t\t\t")},
		{"Error", failureMessage},
		{"Test", tt.Name()},
	}

	msg := message(msgAndArgs...)
	if msg != "" {
		content = append(content, labeledContent{"Messages", msg})
	}
	tt.Errorf("\n%s", ""+labeledOutput(content...))
	return false
}

// isTest tells whether name looks like a test (or benchmark, according to prefix).
// It is a Test (say) if there is a character after Test that is not a lower-case letter.
// We don't want TesticularCancer.
func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}

func message(args ...any) string {
	if len(args) == 0 || args == nil {
		return ""
	}
	if len(args) == 1 {
		msg := args[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(args) > 1 {
		return fmt.Sprintf(args[0].(string), args[1:]...)
	}
	return ""
}

type labeledContent struct {
	label   string
	content string
}

// labeledOutput returns a string consisting of the provided labeledContent.
// Each labeled output is appended in the following manner:
//
//	\t{{label}}:{{align_spaces}}\t{{content}}\n
//
// The initial carriage return is required to undo/erase any padding
// added by testing.T.Errorf. The "\t{{label}}:" is for the label.
// If a label is shorter than the longest label provided, padding
// spaces are added to make all the labels match in length. Once this
// alignment is achieved, "\t{{content}}\n" is added for the output.
//
// If the content of the labeledOutput contains line breaks,
// the subsequent lines are aligned so that they start at
// the same location as the first line.
func labeledOutput(content ...labeledContent) string {
	longestLabel := 0
	for _, v := range content {
		if len(v.label) > longestLabel {
			longestLabel = len(v.label)
		}
	}
	var output string
	for _, v := range content {
		output += "\t" + v.label + ":" +
			strings.Repeat(" ", longestLabel-len(v.label)) + "\t" +
			indentMessageLines(v.content, longestLabel) + "\n"
	}
	return output
}

// Aligns the provided message so that all lines after the first line start at the same location as the first line.
// Assumes that the first line starts at the correct location (after carriage return, tab, label, spacer and tab).
// The longestLabelLen parameter specifies the length of the longest label in the output (required becaues this is the
// basis on which the alignment occurs).
func indentMessageLines(message string, longestLabelLen int) string {
	outBuf := new(bytes.Buffer)

	for i, scanner := 0, bufio.NewScanner(strings.NewReader(message)); scanner.Scan(); i++ {
		// no need to align first line because it starts at the correct location (after the label)
		if i != 0 {
			// append alignLen+1 spaces to align with "{{longestLabel}}:" before adding tab
			outBuf.WriteString("\n\t" + strings.Repeat(" ", longestLabelLen+1) + "\t")
		}
		outBuf.WriteString(scanner.Text())
	}

	return outBuf.String()
}

func buildErrorChainString(err error) string {
	if err == nil {
		return ""
	}

	e := errors.Unwrap(err)
	chain := fmt.Sprintf("%q", err.Error())
	for e != nil {
		chain += fmt.Sprintf("\n\t%q", e.Error())
		e = errors.Unwrap(e)
	}
	return chain
}

// validateEqualArgs checks whether provided arguments can be safely used in the
// Equal/NotEqual functions.
func validateEqualArgs(want, got interface{}) error {
	if want == nil && got == nil {
		return nil
	}

	if isFunction(want) || isFunction(got) {
		return errors.New("cannot take func type as argument")
	}
	return nil
}

// ObjectsAreEqual determines if two objects are considered equal.
//
// This function does no assertion of any kind.
func ObjectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

func isFunction(arg interface{}) bool {
	if arg == nil {
		return false
	}
	return reflect.TypeOf(arg).Kind() == reflect.Func
}

// formatUnequalValues takes two values of arbitrary types and returns string
// representations appropriate to be presented to the user.
//
// If the values are not of like type, the returned strings will be prefixed
// with the type name, and the value will be enclosed in parenthesis similar
// to a type conversion in the Go grammar.
func formatUnequalValues(expected, actual interface{}) (e string, a string) {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		return fmt.Sprintf("%T(%s)", expected, truncatingFormat(expected)),
			fmt.Sprintf("%T(%s)", actual, truncatingFormat(actual))
	}
	switch expected.(type) {
	case time.Duration:
		return fmt.Sprintf("%v", expected), fmt.Sprintf("%v", actual)
	}
	return truncatingFormat(expected), truncatingFormat(actual)
}

// truncatingFormat formats the data and truncates it if it's too long.
//
// This helps keep formatted error messages lines from exceeding the
// bufio.MaxScanTokenSize max line length that the go testing framework imposes.
func truncatingFormat(data interface{}) string {
	value := fmt.Sprintf("%#v", data)
	max := bufio.MaxScanTokenSize - 100 // Give us some space the type info too if needed.
	if len(value) > max {
		value = value[0:max] + "<... truncated>"
	}
	return value
}
