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
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Collection is collection of Variables safe for concurrent use.
type Collection struct {
	mu  sync.RWMutex
	len int64
	db  map[string]Variable
}

// Delete deletes the value for a key.
func (c *Collection) Delete(key string) {
	_, _ = c.LoadAndDelete(key)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c *Collection) Get(k string) (v Variable) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.db[k]
	if !ok {
		return EmptyVariable
	}
	return v
}

// GetWithPrefix return all variables with prefix if any as map[].
func (c *Collection) LoadWithPrefix(prfx string) *Collection {
	vars := new(Collection)
	c.Range(func(v Variable) bool {
		key := v.Key()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			vars.Store(key, v)
		}
		return true
	})
	return vars
}

// Has reprts whether given variable  exists.
func (c *Collection) Has(k string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.db[k]
	return ok
}

// Len of collection.
func (c *Collection) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int(c.len)
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (c *Collection) Load(key string) (v Variable, ok bool) {
	if !c.Has(key) {
		return EmptyVariable, false
	}
	return c.Get(key), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (c *Collection) LoadAndDelete(key string) (v Variable, loaded bool) {
	if !c.Has(key) {
		return EmptyVariable, false
	}

	v = c.Get(key)
	loaded = true
	c.mu.Lock()
	delete(c.db, key)
	atomic.AddInt64(&c.len, -1)
	c.mu.Unlock()
	return
}

// LoadOrDefault returns the existing value for the key if present.
// Much like LoadOrStore, but second argument willl be returned as
// Value whithout being stored into Collection.
func (c *Collection) LoadOrDefault(key string, value any) (v Variable, loaded bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// existing
	if val, ok := c.db[key]; ok {
		return val, true
	}
	if len(key) > 0 {
		if def, ok := value.(Variable); ok {
			return def, false
		}
	}
	v, _ = NewVariable(key, value, false)
	return v, false
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *Collection) LoadOrStore(key string, value any) (actual Variable, loaded bool) {
	k := strings.TrimSpace(key)
	if c.Has(k) {
		return c.Get(key), true
	} else {
		c.Store(k, value)
	}
	return c.Get(key), false
}

// func (c *Collection) MarshalJSON() ([]byte, error) {
// 	pl := make(map[string]any)
// 	c.Range(func(v Variable) bool {
// 		pl[v.Key()] = v.val.raw
// 		return true
// 	})
// 	return json.Marshal(pl)
// }

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (c *Collection) Range(f func(v Variable) bool) {
	c.mu.RLock()
	for _, v := range c.db {
		c.mu.RUnlock()
		f(v)
		c.mu.RLock()
	}
	c.mu.RUnlock()
}

// Store sets the value for a key.
// If it fails to parse val then key will be set to
// EmptyValue with TypeInvalid. Safest would be to
// store Value which error has been already checked.
func (c *Collection) Store(key string, value any) {
	key = strings.TrimSpace(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db == nil {
		c.db = make(map[string]Variable)
	}

	if vv, ok := value.(Variable); ok && vv.Key() == key {
		c.db[key] = vv
	} else {
		c.db[key], _ = NewVariable(key, value, false)
	}

	atomic.AddInt64(&c.len, 1)
}

// ToBytes returns []byte containing
// key = "value"\n.
func (c *Collection) ToBytes() []byte {
	s := c.ToKeyValSlice()
	b := bytes.Buffer{}
	for _, line := range s {
		b.WriteString(line + "\n")
	}
	return b.Bytes()
}

// ToKeyValSlice produces []string slice of strings in format key = "value".
func (c *Collection) ToKeyValSlice() []string {
	r := []string{}
	c.Range(func(v Variable) bool {
		// we can do it directly on interface value since they all are Values
		// implementing Stringer
		r = append(r, fmt.Sprintf("%s=%q", v.Key(), v.String()))
		return true
	})
	return r
}
