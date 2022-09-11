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

package vars

import (
	"errors"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

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
		{"", "", ErrKey},
		{" 0", "", ErrKey},
		{string("\xF4\x8F\xBF\xBF"), "", ErrKey},     // U+10FFFF not printable
		{string("\xF4\x90\x80\x80"), "", ErrKey},     // U+10FFFF+1; out of range
		{string("\xF7\xBF\xBF\xBF"), "", ErrKey},     // 0x1FFFFF; out of range
		{string("\xFB\xBF\xBF\xBF\xBF"), "", ErrKey}, // 0x3FFFFFF; out of range
		{string("\xc0\x80"), "", ErrKey},             // U+0000 encoded in two bytes: incorrect
		{string("\xed\xa0\x80"), "", ErrKey},         // U+D800 high surrogate (sic)
		{string("\xed\xbf\xbf"), "", ErrKey},         // U+DFFF low surrogate (sic)
		{string([]byte{66, 250}), "", ErrKey},
		{string([]byte{66, 250, 67}), "", ErrKey},
		{"aa\xe2", "", ErrKey},
		{"Hello,\"世\"界", "", ErrKey},
		{"\"", "", ErrKey},
		{`"`, "", ErrKey},
		{" ", "", ErrKey},
		{"key=", "", ErrKey},
		{"=key", "", ErrKey},
		{"ke=y", "", ErrKey},
		{"key=key", "", ErrKey},
		{"k\ne\re\t", "", ErrKey},
		{"k\rey", "", ErrKey},
		{"k\ney", "", ErrKey},
		{"k\tey", "", ErrKey},
		{"1key", "", ErrKey},
		{"0123456789key", "", ErrKey},
		{"9key", "", ErrKey},
		{"0key", "", ErrKey},
		{"k\x11ey", "", ErrKey},
		{"$key", "", ErrKey},
		{"key$", "", ErrKey},
		{"key$", "", ErrKey},
		{"key$", "", ErrKey},
		{string([]byte{1, 2, 3, 4, 5}), "", ErrKey},
		{string([]byte{byte('k'), 0xff, byte('e'), 0xfe, byte('y'), 0xfd}), "", ErrKey},
		{string([]byte{byte('k'), byte('e'), byte('y'), 0xff, 0xfe, 0xfd}), "", ErrKey},
		{string("\u07bf"), "", ErrKey},
		{string("A\U000f8500"), "", ErrKey},
		{string("𐀀"), string(rune(65536)), nil}, // 240 144 128 128
	}
}

// parseKeyStd is parseKey equivalent used in tests.
// IMPORTANT! This implementations should reflect optimal key parsing
// done with std libraries so that we can have adequate
// benchmark results for our own implementation.
func parseKeyStd(str string) (key string, err error) {
	if len(str) == 0 {
		return "", ErrKeyIsEmpty
	}

	if !utf8.ValidString(str) {
		return emptyStr, ErrKeyNotValidUTF8
	}

	// remove most outer trimmable characters
	key = strings.TrimFunc(str, func(c rune) bool {
		if c < 256 {
			return autoTrimableKeyChars[c] == 1

		}
		return false
	})

	if len(key) == 0 {
		return "", ErrKeyHasIllegalChar
	}

	if unicode.IsNumber(rune(key[0])) {
		return "", ErrKeyPrefix
	}

	ckey := key
	for len(ckey) > 0 {
		c, size := utf8.DecodeRuneInString(ckey)
		ckey = ckey[size:]
		if unicode.IsControl(c) {
			return "", ErrKeyHasControlChar
		}

		if !unicode.IsPrint(c) {
			return "", ErrKeyHasNonPrintChar
		}
		if c < 256 && (illegalKeyChars[c] == 1) {
			return "", ErrKeyHasIllegalChar
		}
	}
	return key, nil
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
		k, err := ParseKey(test.Key)
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
