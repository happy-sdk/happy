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
)

func FuzzVariableKeys(f *testing.F) {
	for _, test := range getKeyTests() {
		f.Add(test.Key)
	}
	f.Fuzz(func(t *testing.T, arg string) {
		klib, errlib := parseKey(arg)
		kstd, errstd := parseKeyStd(arg)

		if klib != kstd {
			t.Errorf("arg(%s) parsed keys do not match std(%s) != lib(%s)", arg, kstd, klib)
		}

		if (errlib != nil && errstd == nil) || errlib == nil && errstd != nil {
			t.Fatalf("arg(%s) lib error(%v) not like std error(%v)", arg, errlib, errstd)
		}

		// check that key can be recreated
		// and written and read as environment name.
	})
}
