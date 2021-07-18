// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	p := &parser{}
	assert.False(t, p.Flag(0))
	prec, ok := p.Precision()
	assert.False(t, ok)
	assert.Equal(t, 0, prec)
	wid, ok2 := p.Width()
	assert.False(t, ok2)
	assert.Equal(t, 0, wid)
	ret, err := p.Write([]byte{})
	assert.NoError(t, err)
	assert.Equal(t, 0, ret)
}
