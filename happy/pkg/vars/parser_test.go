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
