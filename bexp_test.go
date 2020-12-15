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
		expected := []string{}
		for _, l := range lines {
			if len(l) != 0 {
				expected = append(expected, l[1:len(l)-1])
			}
		}
		result := Expand(cs)
		assert.Equal(t, expected, result)
	}
}

func TestIgnoreDollar(t *testing.T) {
	assert.Equal(t, []string{"${1..3}"}, Expand("${1..3}"))
	assert.Equal(t, []string{"${a,b}${c,d}"}, Expand("${a,b}${c,d}"))
	assert.Equal(t, []string{"x${a,b}x${c,d}x"}, Expand("x${a,b}x${c,d}x"))
}

func TestEmptyOption(t *testing.T) {
	assert.Equal(t, []string{"-v", "-v", "-v", "-v", "-v"}, Expand("-v{,,,,}"))
}

func TestNegativeIncrement(t *testing.T) {
	assert.Equal(t, []string{"3", "2", "1"}, Expand("{3..1}"))
	assert.Equal(t, []string{"10", "9", "8"}, Expand("{10..8}"))
	assert.Equal(t, []string{"10", "09", "08"}, Expand("{10..08}"))
	assert.Equal(t, []string{"c", "b", "a"}, Expand("{c..a}"))

	assert.Equal(t, []string{"4", "2", "0"}, Expand("{4..0..2}"))
	assert.Equal(t, []string{"4", "2", "0"}, Expand("{4..0..-2}"))
	assert.Equal(t, []string{"e", "c", "a"}, Expand("{e..a..2}"))
}

func TestNested(t *testing.T) {
	assert.Equal(t, []string{"a", "b1", "b2", "b3", "c"}, Expand("{a,b{1..3},c}"))
	assert.Equal(t, strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", ""), Expand("{{A..Z},{a..z}}"))
	assert.Equal(t, []string{"ppp", "pppconfig", "pppoe", "pppoeconf"}, Expand("ppp{,config,oe{,conf}}"))
}

func TestOrder(t *testing.T) {
	assert.Equal(t, []string{"ade", "ace", "abe"}, Expand("a{d,c,b}e"))
}

func TestPad(t *testing.T) {
	assert.Equal(t, []string{"9", "10", "11"}, Expand("{9..11}"))
	assert.Equal(t, []string{"09", "10", "11"}, Expand("{09..11}"))
}

func TestSameType(t *testing.T) {
	assert.Equal(t, []string{"{a..9}"}, Expand("{a..9}"))
}

func TestSequence(t *testing.T) {
	//Numeric
	assert.Equal(t, []string{"a1b2c", "a1b3c", "a2b2c", "a2b3c"}, Expand("a{1..2}b{2..3}c"))
	assert.Equal(t, []string{"12", "13", "22", "23"}, Expand("{1..2}{2..3}"))
	//Numeric with step count
	assert.Equal(t, []string{"0", "2", "4", "6", "8"}, Expand("{0..8..2}"))
	assert.Equal(t, []string{"1", "3", "5", "7"}, Expand("{1..8..2}"))
	//Numeric with negative
	assert.Equal(t, []string{"3", "2", "1", "0", "-1", "-2"}, Expand("{3..-2}"))

	//Alphabetic
	assert.Equal(t, []string{"a", "c", "e", "g", "i", "k"}, Expand("{a..k..2}"))
	assert.Equal(t, []string{"b", "d", "f", "h", "j"}, Expand("{b..k..2}"))

}
