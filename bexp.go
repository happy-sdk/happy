// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

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
	escSlash  = "\u0000SLASHbexp1\u0000"
	escOpen   = "\u0000OPENexp1\u0000"
	escClose  = "\u0000CLOSEexp1\u0000"
	escComma  = "\u0000COMMAexp1\u0000"
	escPeriod = "\u0000PERIODexp1\u0000"
)

// BraceExpansion represents bash style brace expansion.
type BraceExpansion []string

// Parse string expresion into BraceExpansion result.
func Parse(str string) BraceExpansion {
	if str == "" {
		return []string{""} // Any incorrectly formed brace expansion is left unchanged.
	}
	// escape a leading {} for case {},a}b / a{},b}c
	if strings.HasPrefix(str, "{}") {
		str = "\\{\\}" + str[2:]
	}
	exp := expand(escapeBraces(str), true)
	return mapArray(exp, unescapeBraces)
}

// ParseValid is for convienience to get errors on input:
// 1. ErrEmptyResult when provided string is empty
// 2. ErrUnchangedBraceExpansion when provided string was left unchanged
// Result will always be `BraceExpansion` with min len 1 to satisfy
// "Any incorrectly formed brace expansion is left unchanged.".
func ParseValid(str string) (BraceExpansion, error) {
	res := Parse(str)
	if len(res) == 1 {
		if err := res.Err(); err != nil {
			return res, err
		}
		if res[0] == str {
			return res, ErrUnchangedBraceExpansion
		}
	}

	return res, nil
}

// MkdirAll calls os.MkdirAll on each math from provided string
// to create a directory tree from brace expansion.
// Error can be ErrEmptyResult if parsing provided str results no paths
// or first error of os.MkdirAll.
func MkdirAll(str string, perm os.FileMode) error {
	if p := Parse(str); p.Err() == nil {
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

// Err returns nil or ErrEmptyResult.
// When working with Brace Expansions this method is for convinience to handle
// only empty string as errors in your program.
// Note that even then it is actually not invalid.
// As Brace Expansion docs say:
// "Any incorrectly formed brace expansion is left unchanged."
// See .ParseValid if you want to get errors if provided string was not
// correctly formed brace expansion.
func (b BraceExpansion) Err() (err error) {
	if len(b) == 0 || (len(b) == 1 && len(b[0]) == 0) {
		err = ErrEmptyResult
	}
	return
}

// Result is convience to get result as string slice.
func (b BraceExpansion) Result() []string {
	return b
}

func parseCommaParts(str string) BraceExpansion {
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

	if regexp.MustCompile(`\$$`).Match([]byte(m.Pre)) {
		expansions := []string{}
		for i := 0; i < len(post); i++ {
			expansions = append(expansions, m.Pre+"{"+m.Body+"}"+post[i])
		}
		return expansions
	}

	isNumericSequence := regexp.MustCompile(`^-?\d+\.\.-?\d+(?:\.\.-?\d+)?$`).Match([]byte(m.Body))
	isAlphaSequence := regexp.MustCompile(`^[a-zA-Z]\.\.[a-zA-Z](?:\.\.-?\d+)?$`).Match([]byte(m.Body))
	isSequence := isNumericSequence || isAlphaSequence
	// isOptions := regexp.MustCompile(`^(.*,)+(.+)?$`).Match([]byte(m.Body))
	isOptions := strings.Contains(m.Body, ",")

	if !isSequence && !isOptions {
		// UseCase???
		if regexp.MustCompile(`,.*\}`).Match([]byte(m.Post)) {
			return expand(m.Pre+"{"+m.Body+escClose+m.Post, false)
		}
		return []string{str}
	}

	var n []string
	var n2 []string

	if isSequence {
		n = strings.Split(m.Body, `..`)
		n2 = expandSequence(n, isAlphaSequence)
	} else {
		n = parseCommaParts(m.Body)
		if len(n) == 1 {
			n = mapArray(expand(n[0], false), embrace)
			if len(n) == 1 {
				return mapArray(post, func(s string) string {
					return m.Pre + n[0] + s
				})
			}
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
	c := strconv.FormatInt(i, 10)
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
	if len(test) == 0 {
		return false
	}

	return unicode.IsNumber(rune(test[0]))
}

func embrace(str string) string {
	return "{" + str + "}"
}

func lte(i int64, y int64) bool {
	return i <= y
}

func gte(i int64, y int64) bool {
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
// 	"\\\\", escSlash,
// 	"\\{", escOpen,
// 	"\\}", escClose,
// 	"\\,", escComma,
// 	"\\.", escPeriod,
// )
//
// escapeBraces.Replace(str)
func escapeBraces(str string) string {

	return sliceAndJoin(
		sliceAndJoin(
			sliceAndJoin(
				sliceAndJoin(
					sliceAndJoin(str, escSlash, "\\\\"), escOpen, "\\{"), escClose, "\\}"), escComma, "\\,"), escPeriod, "\\.")
}

// unescapeBraces is cheaper strings.NewReplacer to escape braces
// var unescapeBraces = strings.NewReplacer(
// 	escSlash, "",
// 	escOpen, "{",
// 	escClose, "}",
// 	escComma, ",",
// 	escPeriod, ".",
// )
func unescapeBraces(str string) string {
	return sliceAndJoin(
		sliceAndJoin(
			sliceAndJoin(
				sliceAndJoin(
					sliceAndJoin(str, "\\", escSlash), "{", escOpen), "}", escClose), ",", escComma), ".", escPeriod)
}

// sliceAndJoin replaces separators
// return strings.Join(strings.Split(str, slice), join).
func sliceAndJoin(str string, join string, slice string) string {
	return strings.ReplaceAll(str, slice, join)
}
