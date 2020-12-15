// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashExpansion(t *testing.T) {
	dat, _ := ioutil.ReadFile("testdata/bash-results.txt")
	cases := strings.Split(string(dat), "><><><><")
	for _, v := range cases {
		lines := strings.Split(v, "\r\n")
		cs := lines[0]
		lines = lines[1:]
		expected := BraceExpansion{}
		for _, l := range lines {
			if len(l) != 0 {
				expected = append(expected, l[1:len(l)-1])
			}
		}
		result := Parse(cs)
		assert.Equal(t, expected, result)
	}
}

func TestIgnoreDollar(t *testing.T) {
	assert.Equal(t, BraceExpansion{"${1..3}"}, Parse("${1..3}"))
	assert.Equal(t, BraceExpansion{"${a,b}${c,d}"}, Parse("${a,b}${c,d}"))
	assert.Equal(t, BraceExpansion{"x${a,b}x${c,d}x"}, Parse("x${a,b}x${c,d}x"))
}

func TestEmptyOption(t *testing.T) {
	assert.Equal(t, BraceExpansion{"-v", "-v", "-v", "-v", "-v"}, Parse("-v{,,,,}"))
}

func TestNegativeIncrement(t *testing.T) {
	assert.Equal(t, BraceExpansion{"3", "2", "1"}, Parse("{3..1}"))
	assert.Equal(t, BraceExpansion{"10", "9", "8"}, Parse("{10..8}"))
	assert.Equal(t, BraceExpansion{"10", "09", "08"}, Parse("{10..08}"))
	assert.Equal(t, BraceExpansion{"c", "b", "a"}, Parse("{c..a}"))

	assert.Equal(t, BraceExpansion{"4", "2", "0"}, Parse("{4..0..2}"))
	assert.Equal(t, BraceExpansion{"4", "2", "0"}, Parse("{4..0..-2}"))
	assert.Equal(t, BraceExpansion{"e", "c", "a"}, Parse("{e..a..2}"))
}

func TestNested(t *testing.T) {
	assert.Equal(t, BraceExpansion{"a", "b1", "b2", "b3", "c"}, Parse("{a,b{1..3},c}"))
	assert.Equal(t, BraceExpansion(strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", "")), Parse("{{A..Z},{a..z}}"))
	assert.Equal(t, BraceExpansion{"ppp", "pppconfig", "pppoe", "pppoeconf"}, Parse("ppp{,config,oe{,conf}}"))
}

func TestOrder(t *testing.T) {
	assert.Equal(t, BraceExpansion{"ade", "ace", "abe"}, Parse("a{d,c,b}e"))
}

func TestPad(t *testing.T) {
	assert.Equal(t, BraceExpansion{"9", "10", "11"}, Parse("{9..11}"))
	assert.Equal(t, BraceExpansion{"09", "10", "11"}, Parse("{09..11}"))
}

func TestSameType(t *testing.T) {
	assert.Equal(t, BraceExpansion{"{a..9}"}, Parse("{a..9}"))
}

func TestSequence(t *testing.T) {
	//Numeric
	assert.Equal(t, BraceExpansion{"a1b2c", "a1b3c", "a2b2c", "a2b3c"}, Parse("a{1..2}b{2..3}c"))
	assert.Equal(t, BraceExpansion{"12", "13", "22", "23"}, Parse("{1..2}{2..3}"))
	//Numeric with step count
	assert.Equal(t, BraceExpansion{"0", "2", "4", "6", "8"}, Parse("{0..8..2}"))
	assert.Equal(t, BraceExpansion{"1", "3", "5", "7"}, Parse("{1..8..2}"))
	//Numeric with negative
	assert.Equal(t, BraceExpansion{"3", "2", "1", "0", "-1", "-2"}, Parse("{3..-2}"))

	//Alphabetic
	assert.Equal(t, BraceExpansion{"a", "c", "e", "g", "i", "k"}, Parse("{a..k..2}"))
	assert.Equal(t, BraceExpansion{"b", "d", "f", "h", "j"}, Parse("{b..k..2}"))

}
