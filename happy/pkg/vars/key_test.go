// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars_test

import (
	"errors"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/mkungla/happy/pkg/vars"
)

var (
	// for faster lookup our custom Unicode Character Table rules
	// we have following two tables.
	keyIllegalChars = [256]uint8{
		'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1,
		'\\': 1, '"': 1, '\'': 1, '`': 1, '=': 1, '$': 1,
	}

	keyAutoTrimableChars = [256]uint8{
		'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1,
		'\\': 1, '"': 1, '\'': 1, '`': 1, ' ': 1,
	}
)

// parseKeyStd is parseKey equivalent used in tests.
// IMPORTANT! This implementations should reflect optimal key parsing
// done with std libraries so that we can have adequate
// benchmark results for our own implementation.
func parseKeyStd(str string) (key string, err error) {
	if len(str) == 0 {
		return "", vars.ErrKeyIsEmpty
	}

	if !utf8.ValidString(str) {
		return "", vars.ErrKeyNotValidUTF8
	}

	// remove most outer trimmable characters
	key = strings.TrimFunc(str, func(c rune) bool {
		if c < 256 {
			return keyAutoTrimableChars[c] == 1

		}
		return false
	})

	if len(key) == 0 {
		return "", vars.ErrKeyHasIllegalChar
	}

	if unicode.IsNumber(rune(key[0])) {
		return "", vars.ErrKeyPrefix
	}

	ckey := key
	for len(ckey) > 0 {
		c, size := utf8.DecodeRuneInString(ckey)
		ckey = ckey[size:]
		if unicode.IsControl(c) {
			return "", vars.ErrKeyHasControlChar
		}

		if !unicode.IsPrint(c) {
			return "", vars.ErrKeyHasNonPrintChar
		}
		if c < 256 && (keyIllegalChars[c] == 1) {
			return "", vars.ErrKeyHasIllegalChar
		}
	}
	return key, nil
}

type keyTest struct {
	Key  string
	Want string
	Err  error
}

func getKeyTests() []keyTest {
	return []keyTest{
		{"key", "key", nil},
		{"key ", "key", nil},
		{" key ", "key", nil},
		{"key\n", "key", nil},
		{"key§", "key§", nil},
		{"k§y", "k§y", nil},
		{"\"key\"", "key", nil},
		{"\" key \"", "key", nil},
		{" k e y ", "k e y", nil},
		{"k e y ", "k e y", nil},
		{" k e y", "k e y", nil},
		{"a1key", "a1key", nil},
		{"_", "_", nil},
		{"###", "###", nil},
		{"§§§", "§§§", nil},
		{" k § y ", "k § y", nil},
		{"\" k § y \"", "k § y", nil},
		{"ýþÿ", "ýþÿ", nil},
		{"keyýþÿ", "keyýþÿ", nil},
		{"kýþÿy", "kýþÿy", nil},
		{" k ý e þ y ÿ 0 ", "k ý e þ y ÿ 0", nil},
		{" k ýþÿ y ", "k ýþÿ y", nil},
		{"\" k ýþÿ y \"", "k ýþÿ y", nil},
		{"ß", "ß", nil},
		{" ß ", "ß", nil},
		{"\"ß\"", "ß", nil},
		{"Hello,世界", "Hello,世界", nil},
		{" Hello,世界 ", "Hello,世界", nil},
		{"\"Hello,世界\"", "Hello,世界", nil},
		{"Hello,世界\"", "Hello,世界", nil},
		{"key\r", "key", nil},
		{"key\n", "key", nil},
		{"key\t", "key", nil},
		{"\rkey", "key", nil},
		{"\nkey", "key", nil},
		{"\tkey", "key", nil},
		{"a0123456789a", "a0123456789a", nil},
		{"Ж", "Ж", nil},
		{"ЖЖ", "ЖЖ", nil},
		{"брэд-ЛГТМ", "брэд-ЛГТМ", nil},
		{"☺☻☹", "☺☻☹", nil},
		{" ☺☻☹ ", "☺☻☹", nil},
		{"a\uFFFDb", "a\uFFFDb", nil},
		{"", "", vars.ErrKey},
		{" 0", "", vars.ErrKey},
		{string("\xF4\x8F\xBF\xBF"), "", vars.ErrKey},     // U+10FFFF not printable
		{string("\xF4\x90\x80\x80"), "", vars.ErrKey},     // U+10FFFF+1; out of range
		{string("\xF7\xBF\xBF\xBF"), "", vars.ErrKey},     // 0x1FFFFF; out of range
		{string("\xFB\xBF\xBF\xBF\xBF"), "", vars.ErrKey}, // 0x3FFFFFF; out of range
		{string("\xc0\x80"), "", vars.ErrKey},             // U+0000 encoded in two bytes: incorrect
		{string("\xed\xa0\x80"), "", vars.ErrKey},         // U+D800 high surrogate (sic)
		{string("\xed\xbf\xbf"), "", vars.ErrKey},         // U+DFFF low surrogate (sic)
		{string([]byte{66, 250}), "", vars.ErrKey},
		{string([]byte{66, 250, 67}), "", vars.ErrKey},
		{"aa\xe2", "", vars.ErrKey},
		{"Hello,\"世\"界", "", vars.ErrKey},
		{"\"", "", vars.ErrKey},
		{`"`, "", vars.ErrKey},
		{" ", "", vars.ErrKey},
		{"key=", "", vars.ErrKey},
		{"=key", "", vars.ErrKey},
		{"ke=y", "", vars.ErrKey},
		{"key=key", "", vars.ErrKey},
		{"k\ne\re\t", "", vars.ErrKey},
		{"k\rey", "", vars.ErrKey},
		{"k\ney", "", vars.ErrKey},
		{"k\tey", "", vars.ErrKey},
		{"1key", "", vars.ErrKey},
		{"0123456789key", "", vars.ErrKey},
		{"9key", "", vars.ErrKey},
		{"0key", "", vars.ErrKey},
		{"k\x11ey", "", vars.ErrKey},
		{"$key", "", vars.ErrKey},
		{"key$", "", vars.ErrKey},
		{"key$", "", vars.ErrKey},
		{"key$", "", vars.ErrKey},
		{string([]byte{1, 2, 3, 4, 5}), "", vars.ErrKey},
		{string([]byte{byte('k'), 0xff, byte('e'), 0xfe, byte('y'), 0xfd}), "", vars.ErrKey},
		{string([]byte{byte('k'), byte('e'), byte('y'), 0xff, 0xfe, 0xfd}), "", vars.ErrKey},
		{string("\u07bf"), "", vars.ErrKey},
		{string("A\U000f8500"), "", vars.ErrKey},
		{string("𐀀"), string(rune(65536)), nil}, // 240 144 128 128
	}
}

func TestParseKeyStdTest(t *testing.T) {
	for _, test := range getKeyTests() {
		key, err := parseKeyStd(test.Key)
		if test.Want != key {
			t.Errorf("in(%s) want(%s) got(%s) err(%v)", test.Key, test.Want, key, err)
		}
		if !errors.Is(err, test.Err) {
			t.Errorf("in(%s) want err(%s) got err(%v)", test.Key, test.Want, err)
		}
	}
}

func TestParseKey(t *testing.T) {
	for _, test := range getKeyTests() {
		// check that key set is correct
		k, err := vars.ParseKey(test.Key)
		if k != test.Want {
			t.Errorf("ParseKey(%s) want key(%s) got key(%v)",
				test.Key, test.Want, k)
		}
		if !errors.Is(err, test.Err) {
			t.Fatalf("ParseKey(%v) want err(%v) got err(%v) k(%s)",
				test.Key, test.Err, err, k)
		}
	}
}

func FuzzVariableKeys(f *testing.F) {
	for _, test := range getKeyTests() {
		f.Add(test.Key)
	}
	f.Fuzz(func(t *testing.T, arg string) {
		klib, errlib := vars.ParseKey(arg)
		kstd, errstd := parseKeyStd(arg)

		if klib != kstd {
			t.Errorf("arg(%s) parsed keys do not match std(%s) != lib(%s)", arg, kstd, klib)
		}

		if (errlib != nil && errstd == nil) || errlib == nil && errstd != nil {
			t.Fatalf("arg(%s) lib error(%v) not like std error(%v)", arg, errlib, errstd)
		}
	})
}
