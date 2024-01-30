// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars

import (
	"strings"
	"unicode"
	"unicode/utf8"
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

// parseKey returns the string key, with all leading
// and trailing illegal characters removed, as defined by unicode
// table for Variable key. based on IEEE Std 1003.1-2001.
// See The Open Group specification for more details.
// https://pubs.opengroup.org/onlinepubs/000095399/basedefs/xbd_chap08.html
func parseKey(str string) (key string, err error) {
	if len(str) == 0 {
		return "", ErrKeyIsEmpty
	}

	if !utf8.ValidString(str) {
		return "", ErrKeyNotValidUTF8
	}

	// remove most outer trimmable characters
	key = strings.TrimFunc(str, func(c rune) bool {
		if c < 256 {
			return keyAutoTrimableChars[c] == 1

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
		if c < 256 && (keyIllegalChars[c] == 1) {
			return "", ErrKeyHasIllegalChar
		}
	}
	return key, nil
}
