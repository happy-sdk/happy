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

package vars_test

// import (
// 	"fmt"
// 	"github.com/mkungla/happy/x/pkg/vars"
// 	"github.com/mkungla/happy/x/pkg/vars/testdata"
// 	"github.com/stretchr/testify/assert"
// 	"math"
// 	"strings"
// 	"testing"
// )

// func FuzzNewValueString(f *testing.F) {
// 	testargs := []string{
// 		"",
// 		"<nil>",
// 		"1",
// 		"0",
// 		"-0",
// 		"-1",
// 		"abc",
// 	}
// 	for _, arg := range testargs {
// 		f.Add(arg)
// 	}
// 	f.Fuzz(func(t *testing.T, arg string) {
// 		v, err := vars.NewValue(arg)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindString, v.Kind())
// 		assert.Equal(t, arg, v.String())
// 		assert.Equal(t, arg, v.Underlying())
// 	})
// }

// func FuzzParseKeyValue(f *testing.F) {
// 	tests := testdata.GetKeyValueParseTests()
// 	for _, test := range tests {
// 		if test.Fuzz {
// 			f.Add(test.Key, test.Val)
// 		}
// 	}
// 	f.Fuzz(func(t *testing.T, key, val string) {
// 		if strings.Contains(key, "=") || val == "=" {
// 			return
// 		}

// 		kv := fmt.Sprintf("%s=%s", key, val)
// 		v, err := vars.ParseVariableFromString(kv)

// 		expkey, _ := vars.ParseKey(key)
// 		expval := testdata.NormalizeExpValue(val)
// 		if err == nil {
// 			assert.Equal(t, vars.KindString, v.Kind())
// 			assert.Equalf(t, expval, v.Underlying(), "val1.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, expval, v.String(), "val1.String -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, expkey, v.Key(), "key1 -> key(%s) val(%s)", key, val)
// 		} else {
// 			assert.Equal(t, vars.KindInvalid, v.Kind())
// 			assert.Equalf(t, nil, v.Underlying(), "val1.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, "", v.String(), "val1.String -> key(%s) val(%s)", key, val)
// 		}

// 		// exceptions and special cases we can not test there
// 		// and should be in TestParseKeyValue
// 		if strings.ContainsRune(val, '"') {
// 			return
// 		}
// 		keyq := fmt.Sprintf("%q", key)
// 		valq := fmt.Sprintf("%q", val)
// 		if strings.Contains(keyq, "\\") || strings.Contains(valq, "\\") {
// 			return
// 		}
// 		kvq := fmt.Sprintf("%s=%s", keyq, valq)
// 		vq, err2 := vars.ParseVariableFromString(kvq)
// 		if err2 == nil {
// 			assert.Equal(t, vars.KindString, vq.Kind())
// 			assert.Equalf(t, val, vq.Underlying(), "val2.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, val, vq.String(), "val2.String -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, expkey, vq.Key(), "key2 -> key(%s) val(%s)", key, val)
// 		} else {
// 			assert.Equal(t, vars.KindInvalid, vq.Kind())
// 			assert.Equalf(t, nil, vq.Underlying(), "val2.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, "", vq.String(), "val2.String -> key(%s) val(%s)", key, val)
// 		}

// 	})
// }
