// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars

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
func parseKey(k string) (key string, err error) {
	var (
		ltrimd bool
		rtrimd bool
	)
	ilen := len(k)
	for i := 0; !ltrimd && i < ilen; i++ {
		ki := k[0]
		if keyAutoTrimableChars[ki] == 1 {
			k = k[1:]
			continue
		}

		if ki >= 48 && ki <= 57 {
			return "", ErrKeyPrefix
		}
		ltrimd = true
	}
	for i := len(k) - 1; !rtrimd && i > 0; i-- {
		if keyAutoTrimableChars[k[i]] == 1 {
			k = k[0:i]
			continue
		}
		rtrimd = true
	}

	n := len(k)
	if n == 0 {
		return "", ErrKeyIsEmpty
	}

	for i := 0; i < n; {
		ki := k[i]

		if ki < utf8RuneSelf {
			i++
			if keyIllegalChars[ki] == 1 {
				return "", ErrKeyHasIllegalChar
			}
			if unicodeIsControl(rune(ki)) {
				return "", ErrKeyHasControlChar
			}
			continue
		}
		x := utf8first[ki]
		if x == xx {
			return "", ErrKeyHasIllegalStarterByte
		}

		size := int(x & 7)
		if i+size > n {
			return "", ErrKeyNotValidUTF8
		}
		accept := utf8AcceptRanges[x>>4]
		if c := k[i+1]; c < accept.lo || accept.hi < c {
			return "", ErrKeyOutOfRange
		} else if size == 2 {
			r := rune(k[i]&mask2)<<6 | rune(k[i+1]&maskx)
			if !unicodeIsPrint(r) {
				return "", ErrKeyHasNonPrintChar
			}
		} else if c := k[i+2]; c < locb || hicb < c {
			return "", ErrKeyNotValidUTF8
		} else if size == 3 {
			r := rune(k[i]&mask3)<<12 | rune(k[i+1]&maskx)<<6 | rune(k[i+2]&maskx)
			if !unicodeIsPrint(r) {
				return "", ErrKeyHasNonPrintChar
			}
		} else if c := k[i+3]; c < locb || hicb < c {
			return "", ErrKeyOutOfRange
		} else if size == 4 {
			r := rune(k[i]&mask4)<<18 | rune(k[i+1]&maskx)<<12 | rune(k[i+2]&maskx)<<6 | rune(k[i+3]&maskx)
			if !unicodeIsPrint(r) {
				return "", ErrKeyHasNonPrintChar
			}
		}
		i += size
	}

	return k, nil
}
