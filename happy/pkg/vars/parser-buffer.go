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
