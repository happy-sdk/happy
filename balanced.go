// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import (
	"regexp"
	"strings"
)

// BalancedResult is returned for the first non-nested matching pair
// of a and b in str.
type BalancedResult struct {
	Valid bool   // is BalancedResult valid
	Start int    // the index of the first match of a
	End   int    // the index of the matching b
	Pre   string // the preamble, a and b not included
	Body  string // the match, a and b not included
	Post  string // the postscript, a and b not included
}

// Balanced returns first non-nested matching pair of a and b in str.
func Balanced(a interface{}, b interface{}, str string) BalancedResult {
	var aVal []byte
	var bVal []byte
	if reg, ok := a.(*regexp.Regexp); ok {
		aVal = maybeMatch(reg, []byte(str))
	} else {
		aVal = []byte(a.(string))
	}
	if reg, ok := b.(*regexp.Regexp); ok {
		bVal = maybeMatch(reg, []byte(str))
	} else {
		bVal = []byte(b.(string))
	}
	return Range(aVal, bVal, str)
}

// Range retruns the first non-nested matching pair of a and b in str.
func Range(a []byte, b []byte, str string) BalancedResult {
	var (
		result []int
		ai     int = -1
		bi     int = -1
	)

	if a != nil {
		ai = strings.Index(str, string(a))
	}

	if b != nil {
		bi = strings.Index(str[ai+1:], string(b))
	}
	if bi != -1 {
		bi += ai + 1
	}
	if ai >= 0 && bi > 0 {
		result = doRange(a, b, ai, bi, str)
	}

	return composeBalancedResult(a, b, str, result)
}

func doRange(a []byte, b []byte, ai, bi int, str string) []int {

	var (
		result []int
		begs   []int

		right int
		left  int
		i     int = ai
	)
	left = len(str)
	for i < len(str) && i >= 0 && result == nil {
		if i == ai {
			begs = append(begs, i)
			ai = strings.Index(str[i+1:], string(a))
			if ai != -1 {
				ai += i + 1
			}
		} else if len(begs) == 1 {
			result = []int{
				begs[len(begs)-1],
				bi,
			}
			begs = begs[:len(begs)-1]
		} else {
			beg := begs[len(begs)-1]
			begs = begs[:len(begs)-1]
			if beg < left {
				left = beg
				right = bi
			}
			bi = strings.Index(str[i+1:], string(b))
			if bi != -1 {
				bi += i + 1
			}
		}
		if ai < bi && ai >= 0 {
			i = ai
		} else {
			i = bi
		}
	}
	if len(begs) > 0 {
		result = []int{
			left,
			right,
		}
	}
	return result
}

func maybeMatch(reg *regexp.Regexp, str []byte) []byte {
	if v := reg.FindAll(str, 1); v != nil {
		return v[0]
	}
	return nil
}

func composeBalancedResult(a []byte, b []byte, str string, result []int) (bres BalancedResult) {
	if len(result) != 2 {
		return
	}
	if result[0]+len(a) < result[1] {
		bres = BalancedResult{
			Valid: true,
			Start: result[0],
			End:   result[1],
			Pre:   str[0:result[0]],
			Body:  str[result[0]+len(a) : result[1]],
			Post:  str[result[1]+len(b):],
		}
	} else {
		bres = BalancedResult{
			Valid: true,
			Start: result[0],
			End:   result[1],
			Pre:   str[0:result[0]],
			Body:  "",
			Post:  str[result[1]+len(b):],
		}
	}
	return
}
