// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2020 The Happy Authors

// Package bexp implements Brace Expansion mechanism to generate arbitrary strings.
// Patterns to be brace expanded take the form of an optional preamble,
// followed by either a series of comma-separated strings or a sequence
// expression between a pair of braces, followed by an optional postscript.
// The preamble is prefixed to each string contained within the braces, and the
// postscript is then appended to each resulting string, expanding left to right.
//
// Brace expansions may be nested. The results of each expanded string are not
// sorted; left to right order is preserved. For example,
//
//	Parse("a{d,c,b}e")
//	[]string{"ade", "ace", "abe"}
//
// Any incorrectly formed brace expansion is left unchanged.
//
// More info about Bash Brace Expansion can be found at
// https://www.gnu.org/software/bash/manual/html_node/Brace-Expansion.html
package bexp

import (
	"errors"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	// ErrEmptyResult representing empty result by parser.
	ErrEmptyResult = errors.New("result is empty")
	// ErrUnchangedBraceExpansion for any incorrectly formed brace expansion
	// where input string is left unchanged.
	ErrUnchangedBraceExpansion = errors.New("brace expansion left unchanged")
)

const (
	// token version.
	tv        = "\u0000v3"
	escSlash  = tv + "SLASH\u0000"
	escOpen   = tv + "OPEN\u0000"
	escClose  = tv + "CLOSE\u0000"
	escComma  = tv + "COMMA\u0000"
	escPeriod = tv + "PERIOD\u0000"
)

// BraceExpansion represents bash style brace expansion.
type BraceExpansion []string

// Parse string expresion into BraceExpansion result.
func Parse(expr string) []string {
	if expr == "" {
		return []string{""}
	}
	// escape a leading {} for case {},a}b / a{},b}c
	if strings.HasPrefix(expr, "{}") {
		expr = "\\{\\}" + expr[2:]
	}

	res := mapArray(expand(escapeBraces(expr), true), unescapeBraces)

	if len(res) == 0 && len(expr) > 0 {
		return []string{""}
	}
	return res
}

// ParseValid is for convienience to get errors on input:
// 1. ErrEmptyResult when provided string is empty
// When working with Brace Expansions this method is for convinience to handle
// only empty string as errors in your program.
// Note that even then it is actually not invalid. As Brace Expansion docs say:
// "Any incorrectly formed brace expansion is left unchanged.".
//
// 2. ErrUnchangedBraceExpansion when provided string was left unchanged
// Result will always be `BraceExpansion` with min len 1 to satisfy
// "Any incorrectly formed brace expansion is left unchanged.".
func ParseValid(expr string) (res []string, err error) {
	res = Parse(expr)

	if len(res) == 1 && res[0] == "" {
		return res, ErrEmptyResult
	}

	if len(res) == 1 && res[0] == expr {
		return res, ErrUnchangedBraceExpansion
	}

	return res, err
}

// MkdirAll calls os.MkdirAll on each math from provided string
// to create a directory tree from brace expansion.
// Error can be ErrEmptyResult if parsing provided str results no paths
// or first error of os.MkdirAll.
func MkdirAll(expr string, perm os.FileMode) error {
	if p, err := ParseValid(expr); err == nil || errors.Is(err, ErrUnchangedBraceExpansion) {
		for _, dir := range p {
			if err := os.MkdirAll(dir, perm); err != nil {
				return err
			}
		}
	}
	return nil
}

// String calls strings.Join(b, " ") and returns resulting string.
func (b BraceExpansion) String() string {
	return strings.Join(b, " ")
}

// Result is convience to get result as string slice.
func (b BraceExpansion) Result() []string {
	return b
}

func parseCommaParts(str string) []string {
	if str == "" {
		return []string{""}
	}
	parts := []string{}
	m := Balanced("{", "}", str)
	if !m.Valid {
		return strings.Split(str, ",")
	}
	pre := m.Pre
	body := m.Body
	post := m.Post
	p := strings.Split(pre, ",")

	p[len(p)-1] += "{" + body + "}"
	postParts := parseCommaParts(post)
	if len(post) > 0 {
		p[len(p)-1] += postParts[0]
		p = append(p, postParts[1:]...)
	}
	parts = append(parts, p...)
	return parts
}

var (
	numericSequenceRegex = regexp.MustCompile(`^-?\d+\.\.-?\d+(?:\.\.-?\d+)?$`)
	alphaSequenceRegex   = regexp.MustCompile(`^[a-zA-Z]\.\.[a-zA-Z](?:\.\.-?\d+)?$`)
	commaInBracesRegex   = regexp.MustCompile(`,.*\}`)
)

func expand(str string, isTop bool) []string {
	m := Balanced("{", "}", str)

	if !m.Valid {
		return []string{str}
	}

	var post []string
	if len(m.Post) > 0 {
		post = expand(m.Post, false)
	} else {
		post = []string{""}
	}

	if strings.HasSuffix(m.Pre, "$") {
		var expansions []string
		for _, p := range post {
			expansions = append(expansions, m.Pre+"{"+m.Body+"}"+p)
		}
		return expansions
	}

	isNumericSequence := numericSequenceRegex.MatchString(m.Body)
	isAlphaSequence := alphaSequenceRegex.MatchString(m.Body)
	isSequence := isNumericSequence || isAlphaSequence
	isOptions := strings.Contains(m.Body, ",")

	if !isSequence && !isOptions {
		if commaInBracesRegex.MatchString(m.Post) {
			return expand(m.Pre+"{"+m.Body+escClose+m.Post, false)
		}
		return []string{str}
	}

	var n, n2 []string
	if isSequence {
		n = strings.Split(m.Body, "..")
		if len(n) == 3 {
			n[2] = strings.TrimLeft(n[2], "0")
		}
		n2 = expandSequence(n, isAlphaSequence)
	} else {
		n = parseCommaParts(m.Body)
		if len(n) == 1 {
			n = mapArray(expand(n[0], false), embrace)
		}
		n2 = concatMap(n, func(el string) []string { return expand(el, false) })
	}

	return expandToExpansionSlice(n2, post, m.Pre, isTop, isSequence)
}

func expandSequence(n []string, isAlphaSequence bool) []string {
	x := numeric(n[0]) //nolint: ifshort
	y := numeric(n[1]) //nolint: ifshort
	width := max(len(n[0]), len(n[1]))

	var incr int64
	if len(n) == 3 {
		incr = int64(math.Abs(float64(numeric(n[2]))))
	} else {
		incr = 1
	}

	test := lte
	// reverse
	if y < x {
		incr *= -1
		test = gte
	}

	pad := some(n, isPadded)

	n2 := []string{}

	for i := x; test(i, y); i += incr {
		var c string
		if isAlphaSequence {
			c = string(rune(i))
			if c == "\\" {
				c = ""
			}
		} else {
			c = expandNonAlphaSequence(i, width, pad)
		}
		n2 = append(n2, c)
	}
	return n2
}

func expandNonAlphaSequence(i int64, width int, pad bool) string {
	c := strconv.FormatInt(i, 10) //nolint: gomnd
	if pad {
		var need = width - len(c)
		if need > 0 {
			var z = strings.Join(make([]string, need+1), "0")
			if i < 0 {
				c = "-" + z + c[1:]
			} else {
				c = z + c
			}
		}
	}
	return c
}

func expandToExpansionSlice(n2, post []string, pre string, isTop, isSequence bool) []string {
	expansions := []string{}
	for j := 0; j < len(n2); j++ {
		for k := 0; k < len(post); k++ {
			expansion := pre + n2[j] + post[k]
			if !isTop || isSequence || expansion != "" {
				expansions = append(expansions, expansion)
			}
		}
	}
	return expansions
}

func concatMap(xs []string, fn func(el string) []string) []string {
	res := []string{}
	for i := 0; i < len(xs); i++ {
		var x = fn(xs[i])
		res = append(res, x...)
	}
	return res
}

func some(arr []string, fn func(el string) bool) bool {
	for _, v := range arr {
		if fn(v) {
			return true
		}
	}
	return false
}

func mapArray(arr []string, call func(str string) string) []string {
	ret := []string{}
	for _, v := range arr {
		ret = append(ret, call(v))
	}
	return ret
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// isPadded reports whether string starts zero left padded number.
// e.g (-?) 00, 01, 0001, -01 etc.
// Current implementation is dropin replacement to
// regexp.MustCompile(`^-?0\d`).Match([]byte(el)) and is about 15x faster.
func isPadded(el string) bool {
	if len(el) < 2 {
		return false
	}
	if !strings.HasPrefix(el, "0") && !strings.HasPrefix(el, "-0") {
		return false
	}
	test := strings.TrimLeft(el, "-")[1:]
	if test == "" {
		return false
	}

	return unicode.IsNumber(rune(test[0]))
}

func embrace(str string) string {
	return "{" + str + "}"
}

func lte(i, y int64) bool {
	return i <= y
}

func gte(i, y int64) bool {
	return i >= y
}

func numeric(str string) int64 {
	v, err := strconv.Atoi(str)
	if err != nil {
		return int64(str[0])
	}
	return int64(v)
}

// escapeBraces is cheaper strings.NewReplacer to escape braces
//
// var escapeBraces = strings.NewReplacer(
//
//	"\\\\", escSlash,
//	"\\{", escOpen,
//	"\\}", escClose,
//	"\\,", escComma,
//	"\\.", escPeriod,
//
// )
//
// escapeBraces.Replace(str).
func escapeBraces(str string) string {
	return sliceAndJoin(
		sliceAndJoin(
			sliceAndJoin(
				sliceAndJoin(
					sliceAndJoin(str, escSlash, "\\\\"), escOpen, "\\{"), escClose, "\\}"), escComma, "\\,"), escPeriod, "\\.")
}

// unescapeBraces is cheaper strings.NewReplacer to escape braces
// var unescapeBraces = strings.NewReplacer(
//
//	escSlash, "",
//	escOpen, "{",
//	escClose, "}",
//	escComma, ",",
//	escPeriod, ".",
//
// ).
func unescapeBraces(str string) string {
	return sliceAndJoin(
		sliceAndJoin(
			sliceAndJoin(
				sliceAndJoin(
					sliceAndJoin(str, "\\", escSlash), "{", escOpen), "}", escClose), ",", escComma), ".", escPeriod)
}

// sliceAndJoin replaces separators
// return strings.Join(strings.Split(str, slice), join).
func sliceAndJoin(str, join, slice string) string {
	return strings.ReplaceAll(str, slice, join)
}
