// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import "unicode/utf8"

func (b *parserBuffer) write(p []byte) {
	*b = append(*b, p...)
}

func (b *parserBuffer) writeString(s string) {
	*b = append(*b, s...)
}

func (b *parserBuffer) writeByte(c byte) {
	*b = append(*b, c)
}

func (b *parserBuffer) writeRune(r rune) {
	if r < utf8.RuneSelf {
		*b = append(*b, byte(r))
		return
	}

	bb := *b
	n := len(bb)
	for n+utf8.UTFMax > cap(bb) {
		bb = append(bb, 0)
	}
	w := utf8.EncodeRune(bb[n:n+utf8.UTFMax], r)
	*b = bb[:n+w]
}
