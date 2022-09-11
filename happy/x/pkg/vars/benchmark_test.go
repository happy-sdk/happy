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

import (
	"fmt"
	"github.com/mkungla/happy/x/pkg/vars"
	"math"
	"sync"
	"testing"
)

func BenchmarkVariableMapSingleValue(b *testing.B) {
	key, value := "key", "value"
	b.Run("vars.Map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := new(vars.Map)
			m.Store(key, value)

			v, ok := m.Load(key)
			if !ok || v.String() != value {
				b.Fatal("bad value")
			}
		}
	})

	b.Run("sync.Map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := new(sync.Map)
			m.Store(key, value)

			v, ok := m.Load(key)
			if !ok || v.(string) != value {
				b.Fatal("bad value")
			}
		}
	})
}

type MapTest struct {
	Key           string // key
	Value         any    // value
	ValueString   string // value
	InitiallySet  bool   // stored immedietly
	Delete        bool
	LoadAndDelete bool
}

func BenchmarkVariableMaps(b *testing.B) {
	var tests = []MapTest{
		{"GO111MODULE", "", "", true, false, false},
		{"GOARCH", "amd64", "amd64", true, false, true},
		{"key", "value", "value", true, false, true},
		{"bool", true, "true", true, false, true},
		{"int", math.MaxInt64, fmt.Sprint(math.MaxInt64), true, false, true},
		{"float", math.MaxFloat64, fmt.Sprint(math.MaxFloat64), true, false, true},
		{"GOCACHE", "/home/user/.cache/go-build", "/home/user/.cache/go-build", true, false, false},
		{"GOPATH", "/home/user/go", "/home/user/go", false, false, false},
		{"GOTOOLDIR", "/usr/lib/golang/pkg/tool/linux_amd64", "/usr/lib/golang/pkg/tool/linux_amd64", true, true, false},
		{"CGO_ENABLED", 1, "1", true, false, false},
	}

	var wantLen int
	for _, t := range tests {
		if t.Delete || t.LoadAndDelete {
			continue
		}
		wantLen++
	}

	b.Run("vars.Map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var libLen int
			m := new(vars.Map)
			// InitiallySet
			for _, t := range tests {
				if !t.InitiallySet {
					continue
				}
				m.Store(t.Key, t.Value)
				if v, ok := m.Load(t.Key); !ok || t.ValueString != v.String() {
					b.Errorf("Store: %v", t)
				}
			}

			// LoadOrStore
			for _, t := range tests {
				if t.InitiallySet {
					continue
				}
				v, ok := m.LoadOrStore(t.Key, t.Value)
				if ok || t.ValueString != v.String() {
					b.Errorf("LoadOrStore: %v", t)
				}
			}

			// Delete
			for _, t := range tests {
				if !t.Delete {
					continue
				}
				m.Delete(t.Key)
				if _, ok := m.Load(t.Key); ok {
					b.Errorf("Delete: %v not deleted", t)
				}
			}

			// LoadAndDelete
			for _, t := range tests {
				if !t.LoadAndDelete {
					continue
				}
				m.LoadAndDelete(t.Key)
				if _, ok := m.Load(t.Key); ok {
					b.Errorf("Delete: %v not deleted", t)
				}
			}

			// Range
			m.Range(func(v vars.Variable) bool {
				libLen++
				_ = v.Key()
				_ = v.String()
				return true
			})
			if libLen != wantLen {
				b.Errorf("After range want len: %d got %d", wantLen, libLen)
			}
		}
	})

	b.Run("sync.Map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var stdLen int
			m := new(sync.Map)
			// InitiallySet
			for _, t := range tests {
				if !t.InitiallySet {
					continue
				}
				m.Store(t.Key, t.Value)
				v, ok := m.Load(t.Key)
				val := fmt.Sprint(v)
				if !ok || t.ValueString != val {
					b.Errorf("Store: %v", t)
				}
			}

			// LoadOrStore
			for _, t := range tests {
				if t.InitiallySet {
					continue
				}
				v, ok := m.LoadOrStore(t.Key, t.Value)
				val := fmt.Sprint(v)
				if ok || t.ValueString != val {
					b.Errorf("LoadOrStore: %v", t)
				}
			}

			// Delete
			for _, t := range tests {
				if !t.Delete {
					continue
				}
				m.Delete(t.Key)
				if _, ok := m.Load(t.Key); ok {
					b.Errorf("Delete: %v not deleted", t)
				}
			}

			// LoadAndDelete
			for _, t := range tests {
				if !t.LoadAndDelete {
					continue
				}
				m.LoadAndDelete(t.Key)
				if _, ok := m.Load(t.Key); ok {
					b.Errorf("Delete: %v not deleted", t)
				}
			}
			m.Range(func(key, value any) bool {
				stdLen++
				_ = key.(string)
				_ = fmt.Sprint(value)
				return true
			})
			if stdLen != wantLen {
				b.Errorf("After range want len: %d got %d", wantLen, stdLen)
			}
		}
	})
}
