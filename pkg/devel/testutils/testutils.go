// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package testutils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
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

// Assert checks that the specified condition is true, failing the test if false.
//
// Example:
//
//	testutils.Assert(t, x > 0, "x should be positive, got %d", x)
func Assert(tt TestingIface, condition bool, msgAndArgs ...any) bool {
	tt.Helper()
	if !condition {
		return fail(tt, "Assertion failed", msgAndArgs...)
	}
	return true
}

// True asserts that the specified value is true.
//
// Deprecated: Use Assert instead.
// Replace:
//
//	testutils.True(t, myBool, "myBool should be true")
//
// With:
//
//	testutils.Assert(t, myBool, "myBool should be true")
func True(tt TestingIface, value bool, msgAndArgs ...any) bool {
	tt.Helper()
	return Assert(tt, value, msgAndArgs...)
}

// False asserts that the specified value is false.
//
// Deprecated: Use Assert with negation instead.
// Replace:
//
//	testutils.False(t, myBool, "myBool should be false")
//
// With:
//
//	testutils.Assert(t, !myBool, "myBool should be false")
func False(tt TestingIface, value bool, msgAndArgs ...any) bool {
	tt.Helper()
	return Assert(tt, !value, msgAndArgs...)
}

// NotNil asserts that the specified value is not nil.
//
//	testutils.NotNil(t, &val)
func NotNil(tt TestingIface, value any, msgAndArgs ...any) bool {
	if isNil(value) {
		tt.Helper()
		return fail(tt, "Should not be <nil>", msgAndArgs...)
	}
	return true
}

// Nil asserts that the specified value is nil.
func Nil(tt TestingIface, value any, msgAndArgs ...any) bool {
	if !isNil(value) {
		tt.Helper()
		return fail(tt, "Should be <nil>", msgAndArgs...)
	}
	return true
}

// isNil checks if a value is nil, handling both untyped nil and typed nil pointers/interfaces
func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
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
func validateEqualArgs(want, got any) error {
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
func ObjectsAreEqual(expected, actual any) bool {
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

func isFunction(arg any) bool {
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
func formatUnequalValues(expected, actual any) (e string, a string) {
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
func truncatingFormat(data any) string {
	value := fmt.Sprintf("%#v", data)
	max := bufio.MaxScanTokenSize - 100 // Give us some space the type info too if needed.
	if len(value) > max {
		value = value[0:max] + "<... truncated>"
	}
	return value
}

func ExtractCoverage(s string) (string, error) {
	// Match coverage percentage
	covRe := regexp.MustCompile(`coverage: (\d+\.\d+%)`)
	if match := covRe.FindStringSubmatch(s); len(match) > 1 {
		return match[1], nil
	}

	fields := strings.Fields(s)
	if len(fields) >= 3 {
		return fields[2], nil
	}

	// Match "no test files"
	noTestRe := regexp.MustCompile(`\[no test files\]`)
	if noTestRe.MatchString(s) {
		return "no test files", nil
	}

	return "", fmt.Errorf("no coverage info found")
}

// Containable defines types that can be searched for containment
type Containable[T comparable] interface {
	~[]T | ~string | ~map[T]any
}

// Contains is the unified generic method that dispatches to the correct implementation
func Contains[Container Containable[Item], Item comparable](
	tt TestingIface,
	container Container,
	item Item,
	msgAndArgs ...any,
) bool {
	tt.Helper()
	return contains(tt, container, item, msgAndArgs...)
}

// Alternative: Separate interfaces for better type safety
type SliceOrMap[T comparable] interface {
	~[]T | ~map[T]any
}

// ContainsExact for exact matching (slices and map keys)
func ContainsExact[Container SliceOrMap[Item], Item comparable](
	tt TestingIface,
	container Container,
	item Item,
	msgAndArgs ...any,
) bool {
	tt.Helper()
	return containsExact(tt, container, item, msgAndArgs...)
}

func containsExact[Container SliceOrMap[Item], Item comparable](
	tt TestingIface,
	container Container,
	item Item,
	msgAndArgs ...any,
) bool {
	tt.Helper()

	switch c := any(container).(type) {
	case []Item:
		if slices.Contains(c, item) {
			return true
		}
		return fail(tt, fmt.Sprintf("Slice does not contain item:\n"+
			"slice: %v\n"+
			"item: %v", c, item), msgAndArgs...)

	case map[Item]any:
		if _, exists := c[item]; exists {
			return true
		}
		return fail(tt, fmt.Sprintf("Map does not contain key:\n"+
			"map keys: %v\n"+
			"key: %v", maps.Keys(c), item), msgAndArgs...)

	default:
		return fail(tt, fmt.Sprintf("Unsupported container type: %T", container), msgAndArgs...)
	}
}

func contains[Container Containable[Item], Item comparable](
	tt TestingIface,
	container Container,
	item Item,
	msgAndArgs ...any,
) bool {
	tt.Helper()

	// Use type switching on the actual value to dispatch
	switch c := any(container).(type) {
	case []Item:
		if len(c) == 0 {
			return fail(tt, fmt.Sprintf("Slice is empty so can not contain item:\n"+
				"slice: %v\n"+
				"item: %v", c, item), msgAndArgs...)
		}
		// Slice contains - use slices.Contains
		if slices.Contains(c, item) {
			return true
		}
		return fail(tt, fmt.Sprintf("Slice does not contain item:\n"+
			"slice: %v\n"+
			"item: %v", c, item), msgAndArgs...)

	case string:
		return fail(tt, "use ContainsString instead")
	case map[Item]any:
		// Map key contains
		if _, exists := c[item]; exists {
			return true
		}
		return fail(tt, fmt.Sprintf("Map does not contain key:\n"+
			"map keys: %v\n"+
			"key: %v", maps.Keys(c), item), msgAndArgs...)

	default:
		return fail(tt, fmt.Sprintf("Unsupported container type for Contains: %T", container), msgAndArgs...)
	}
}

// ContainsFunc provides functional matching for slices (can't be unified easily)
func ContainsFunc[T any](tt TestingIface, container []T, predicate func(T) bool, msgAndArgs ...any) bool {
	tt.Helper()
	if slices.ContainsFunc(container, predicate) {
		return true
	}
	return fail(tt, fmt.Sprintf("Slice does not contain element matching predicate:\n"+
		"slice: %v", container), msgAndArgs...)
}

// ContainsValue for checking map values (separate since it's a different operation)
func ContainsValue[K comparable, V comparable](tt TestingIface, m map[K]V, value V, msgAndArgs ...any) bool {
	tt.Helper()
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return fail(tt, fmt.Sprintf("Map does not contain value:\n"+
		"map: %v\n"+
		"value: %v", m, value), msgAndArgs...)
}

// More specific versions are still available if needed for clarity
func ContainsSlice[T comparable](tt TestingIface, slice []T, item T, msgAndArgs ...any) bool {
	return contains(tt, slice, item, msgAndArgs...)
}

func ContainsString(tt TestingIface, str, substr string, msgAndArgs ...any) bool {
	tt.Helper()
	if str == "" {
		return fail(tt, fmt.Sprintf("String is empty so can not contain substring:\n"+
			"string: %q\n"+
			"substring: %v", str, substr), msgAndArgs...)
	}
	if strings.Contains(str, substr) {
		return true
	}
	return fail(tt, fmt.Sprintf("String does not contain substring:\n"+
		"string: %q\n"+
		"substring: %v", str, substr), msgAndArgs...)
}

func ContainsKey[K comparable, V any](tt TestingIface, m map[K]V, key K, msgAndArgs ...any) bool {
	tt.Helper()
	// Convert to map[K]any for the generic Contains
	anyMap := make(map[K]any, len(m))
	for k, v := range m {
		anyMap[k] = v
	}
	return Contains(tt, anyMap, key, msgAndArgs...)
}

func ContainsWithEq[T any](tt TestingIface, container []T, item T, eq func(T, T) bool, msgAndArgs ...any) bool {
	tt.Helper()
	for _, v := range container {
		if eq(v, item) {
			return true
		}
	}
	return fail(tt, fmt.Sprintf("Slice does not contain item with custom equality:\n"+
		"slice: %v\n"+
		"item: %v", container, item), msgAndArgs...)
}

// IsType asserts that the given value is of the expected type. If the value is
// not of the expected type, the test fails with a descriptive message.
//
// Example:
//
//	var x interface{} = 42
//	testutils.IsType(t, 0, x, "Expected int type")
//	// Fails if x is not an int
func IsType(tt TestingIface, expectedType any, got any, msgAndArgs ...any) bool {
	t.Helper()
	expected := reflect.TypeOf(expectedType)
	if expected == nil {
		if got != nil {
			fail(t, fmt.Sprintf("expected nil type, got %T", got), msgAndArgs...)
			return false
		}
		return true
	}
	gotType := reflect.TypeOf(got)
	if gotType != expected {
		fail(t, fmt.Sprintf("expected type %v, got %v", expected, gotType), msgAndArgs...)
	}
	return gotType == expected
}
