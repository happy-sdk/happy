// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"fmt"
	"strconv"
	"strings"
)

func parseBool(str string) (r bool, s string, e error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		r, s = true, "true"
	case "0", "f", "F", "false", "FALSE", "False":
		r, s = false, "false"
	default:
		r, s, e = false, "", strconv.ErrSyntax
	}
	return r, s, e
}

func parseFloat(str string, bitSize int) (r float64, s string, e error) {
	r, e = strconv.ParseFloat(str, bitSize)
	// s = strconv.FormatFloat(r, 'f', -1, bitSize)
	if bitSize == 32 {
		s = fmt.Sprintf("%v", float32(r))
	} else {
		s = fmt.Sprintf("%v", r)
	}
	return r, s, e
}

func parseComplex64(str string) (r complex64, s string, e error) {
	fields := strings.Fields(str)
	if len(fields) != 2 {
		return complex64(0), "", strconv.ErrSyntax
	}
	var err error
	var f1, f2 float32
	var s1, s2 string
	lf1, s1, err := parseFloat(fields[0], 32)
	if err != nil {
		return complex64(0), "", err
	}
	f1 = float32(lf1)

	rf2, s2, err := parseFloat(fields[1], 32)
	if err != nil {
		return complex64(0), "", err
	}
	f2 = float32(rf2)
	s = s1 + " " + s2
	r = complex64(complex(f1, f2))
	return r, s, e
}

func parseComplex128(str string) (r complex128, s string, e error) {
	fields := strings.Fields(str)
	if len(fields) != 2 {
		return complex128(0), "", strconv.ErrSyntax
	}
	var err error
	var f1, f2 float64
	var s1, s2 string
	lf1, s1, err := parseFloat(fields[0], 64)
	if err != nil {
		return complex128(0), "", err
	}
	f1 = float64(lf1)

	rf2, s2, err := parseFloat(fields[1], 64)
	if err != nil {
		return complex128(0), "", err
	}
	f2 = float64(rf2)
	s = s1 + " " + s2
	r = complex128(complex(f1, f2))
	return r, s, e
}

func parseInt(str string, base, bitSize int) (r int64, s string, e error) {
	r, e = strconv.ParseInt(str, base, bitSize)
	s = strconv.Itoa(int(r))
	return r, s, e
}

func parseUint(str string, base, bitSize int) (r uint64, s string, e error) {
	r, e = strconv.ParseUint(str, base, bitSize)
	s = strconv.Itoa(int(r))
	return r, s, e
}
