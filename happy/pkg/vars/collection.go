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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Collection is like a Go sync.Map safe for concurrent use
// by multiple goroutines without additional locking or coordination.
// Loads, stores, and deletes run in amortized constant time.
//
// The zero Map is empty and ready for use.
// A Map must not be copied after first use.
type Collection struct {
	mu  sync.RWMutex
	len int64
	db  map[string]Value
}

// Store stores the variable for a variable.Key().
func (c *Collection) Store(key string, value any) {
	key = strings.TrimSpace(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db == nil {
		c.db = make(map[string]Value)
	}
	if vv, ok := value.(Value); ok {
		c.db[key] = vv
	} else {
		c.db[key] = NewValue(value)
	}
	atomic.AddInt64(&c.len, 1)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *Collection) LoadOrStore(key string, value any) (actual Value, loaded bool) {
	k := strings.TrimSpace(key)
	if c.Has(k) {
		return c.Get(key), true
	} else {
		c.Store(k, value)
	}
	return
}

// Set adds key value pair into collection. if not already set.
func (c *Collection) Set(k string, v any) {
	c.LoadOrStore(k, v)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c *Collection) Get(k string) (val Value) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.db[k]
	if !ok {
		return EmptyValue
	}
	return val
}

// Getd returns Value by key if exists otherwise
// it returns Value representing defval argument.
func (c *Collection) Getd(k string, defval any) (val Value) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.db[k]
	if !ok {
		if len(k) > 0 {
			if def, ok := defval.(Value); ok {
				return def
			}
			d := NewValue(defval)
			return d
		}
		e := NewValue("")
		return e
	}
	return val
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
func (c *Collection) Range(f func(key string, value Value) bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for key, value := range c.db {
		f(key, value)
	}
}

// GetWithPrefix return all variables with prefix if any as map[].
func (c *Collection) GetWithPrefix(prfx string) *Collection {
	vars := new(Collection)
	c.Range(func(key string, value Value) bool {
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			vars.Set(key, value)
		}
		return true
	})
	return vars
}

// ToKeyValSlice produces []string slice of strings in format key = "value".
func (c *Collection) ToKeyValSlice() []string {
	r := []string{}
	c.Range(func(key string, value Value) bool {
		// we can do it directly on interface value since they all are Values
		// implementing Stringer
		r = append(r, fmt.Sprintf("%s=%q", key, value))
		return true
	})
	return r
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

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (c *Collection) Load(key string) (value Value, ok bool) {
	if !c.Has(key) {
		return EmptyValue, false
	}
	return c.Get(key), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (c *Collection) LoadAndDelete(key string) (value Value, loaded bool) {
	if !c.Has(key) {
		return EmptyValue, false
	}

	value = c.Get(key)
	c.mu.Lock()
	atomic.AddInt64(&c.len, -1)
	c.mu.Unlock()
	return
}

// Delete deletes the value for a key.
func (c *Collection) Delete(key string) {
	_, _ = c.LoadAndDelete(key)
}

func (c *Collection) MarshalJSON() ([]byte, error) {
	pl := make(map[string]any)
	c.Range(func(key string, value Value) bool {
		pl[key] = value.raw
		return true
	})
	return json.Marshal(pl)
}
