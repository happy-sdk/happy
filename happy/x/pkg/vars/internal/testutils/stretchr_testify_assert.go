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

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...interface{})
}

type tHelper interface {
	Helper()
}

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.NoError(t, err) {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoError(t TestingT, err error, msgAndArgs ...interface{}) bool {
	if err != nil {
		if h, ok := t.(tHelper); ok {
			h.Helper()
		}
		return Fail(t, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
	}

	return true
}

// Equal asserts that two objects are equal.
//
//	assert.Equal(t, 123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equal[V comparable](t TestingT, expected, actual V, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if expected != actual {
		return Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %v\n"+
			"actual  : %v", expected, actual), msgAndArgs...)
	}
	return true
}

// Equalf asserts that two objects are equal.
//
//	assert.Equalf(t, 123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equalf[V comparable](t TestingT, expected V, actual V, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Equal(t, expected, actual, append([]any{msg}, args...)...)
}

func EqualAny(t TestingT, expected, actual any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if err := validateEqualArgs(expected, actual); err != nil {
		return Fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (%s)",
			expected, actual, err), msgAndArgs...)
	}

	if !ObjectsAreEqual(expected, actual) {
		expected, actual = formatUnequalValues(expected, actual)
		return Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s", expected, actual), msgAndArgs...)
	}

	return true

}

func EqualAnyf(t TestingT, expected any, actual any, msg string, args ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return EqualAny(t, expected, actual, append([]any{msg}, args...)...)
}

func ErrorIs(t TestingT, err, target error, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if errors.Is(err, target) {
		return true
	}

	var expectedText string
	if target != nil {
		expectedText = target.Error()
	}

	chain := buildErrorChainString(err)

	return Fail(t, fmt.Sprintf("Target error should be in err chain:\n"+
		"expected: %q\n"+
		"in chain: %s", expectedText, chain,
	), msgAndArgs...)
}

func Error(t TestingT, err error, msgAndArgs ...interface{}) bool {
	if err == nil {
		if h, ok := t.(tHelper); ok {
			h.Helper()
		}
		return Fail(t, "An error is expected but got nil.", msgAndArgs...)
	}

	return true
}

// True asserts that the specified value is true.
//
//	assert.True(t, myBool)
func True(t TestingT, value bool, msgAndArgs ...interface{}) bool {
	if !value {
		if h, ok := t.(tHelper); ok {
			h.Helper()
		}
		return Fail(t, "Should be true", msgAndArgs...)
	}

	return true

}

// False asserts that the specified value is false.
//
//	assert.False(t, myBool)
func False(t TestingT, value bool, msgAndArgs ...interface{}) bool {
	if value {
		if h, ok := t.(tHelper); ok {
			h.Helper()
		}
		return Fail(t, "Should be false", msgAndArgs...)
	}

	return true

}

func Fail(t TestingT, failureMessage string, msgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	content := []labeledContent{
		{"Error Trace", strings.Join(CallerInfo(), "\n\t\t\t")},
		{"Error", failureMessage},
	}

	// Add test name if the Go version supports it
	if n, ok := t.(interface {
		Name() string
	}); ok {
		content = append(content, labeledContent{"Test", n.Name()})
	}

	message := messageFromMsgAndArgs(msgAndArgs...)
	if len(message) > 0 {
		content = append(content, labeledContent{"Messages", message})
	}

	t.Errorf("\n%s", ""+labeledOutput(content...))

	return false
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

// validateEqualArgs checks whether provided arguments can be safely used in the
// Equal/NotEqual functions.
func validateEqualArgs(expected, actual interface{}) error {
	if expected == nil && actual == nil {
		return nil
	}

	if isFunction(expected) || isFunction(actual) {
		return errors.New("cannot take func type as argument")
	}
	return nil
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

func isFunction(arg interface{}) bool {
	if arg == nil {
		return false
	}
	return reflect.TypeOf(arg).Kind() == reflect.Func
}

type labeledContent struct {
	label   string
	content string
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

func messageFromMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}

// labeledOutput returns a string consisting of the provided labeledContent. Each labeled output is appended in the following manner:
//
//	\t{{label}}:{{align_spaces}}\t{{content}}\n
//
// The initial carriage return is required to undo/erase any padding added by testing.T.Errorf. The "\t{{label}}:" is for the label.
// If a label is shorter than the longest label provided, padding spaces are added to make all the labels match in length. Once this
// alignment is achieved, "\t{{content}}\n" is added for the output.
//
// If the content of the labeledOutput contains line breaks, the subsequent lines are aligned so that they start at the same location as the first line.
func labeledOutput(content ...labeledContent) string {
	longestLabel := 0
	for _, v := range content {
		if len(v.label) > longestLabel {
			longestLabel = len(v.label)
		}
	}
	var output string
	for _, v := range content {
		output += "\t" + v.label + ":" + strings.Repeat(" ", longestLabel-len(v.label)) + "\t" + indentMessageLines(v.content, longestLabel) + "\n"
	}
	return output
}

/* CallerInfo is necessary because the assert functions use the testing object
internally, causing it to print the file:line of the assert method, rather than where
the problem actually occurred in calling code.*/

// CallerInfo returns an array of strings containing the file and line number
// of each stack frame leading from the current test to the assert call that
// failed.
func CallerInfo() []string {

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

// Stolen from the `go test` tool.
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
