// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vars

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Set updates key value pair in collection. If key does not exist then appends
// key wth given value
func (c *Collection) Set(k string, v interface{}) {
	k = strings.TrimSpace(k)
	vv, _ := NewValue(v)
	c.m.Store(k, &vv)
	atomic.AddInt64(&c.len, 1)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c *Collection) Get(k string, defval ...interface{}) (val *Value) {
	value, ok := c.m.Load(k)
	val, valid := value.(*Value)
	if len(k) == 0 || !ok || !valid || val.Len() == 0 {
		if len(defval) > 0 {
			d, _ := NewValue(defval[0])
			return &d
		}
    if !valid {
      e, _ := NewValue("")
      return &e
    }
	}
	return val
}

// Has reprts whether given variable  exists
func (c *Collection) Has(k string) bool {
	_, ok := c.m.Load(k)
	return ok
}

// Len of collection
func (c *Collection) Len() int {
	return int(c.len)
}

// GetWithPrefix return all variables with prefix if any as map[]
func (c *Collection) GetWithPrefix(prfx string) (vars Collection) {
	vars = Collection{m: &sync.Map{}}
	c.Range(func(key string, value *Value) bool {
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			vars.Set(key, value)
		}
		return true
	})
	return
}

// ToKeyValSlice produces []string slice of strings in format key = "value"
func (c *Collection) ToKeyValSlice() []string {
	r := []string{}
	c.m.Range(func(key, value interface{}) bool {
		r = append(r, fmt.Sprintf("%s=%q", key, value))
		return true
	})
	return r
}

// ToBytes returns []byte containing
// key = "value"\n
func (c *Collection) ToBytes() []byte {
	s := c.ToKeyValSlice()
	b := bytes.Buffer{}
	for _, line := range s {
		b.WriteString(line + "\n")
	}
	return b.Bytes()
}

// Store stores the variable for a variable.Key().
func (c *Collection) Store(key string, value interface{}) {
  v, _ := NewValue(value)
  c.m.Store(key, &v)
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (c *Collection) Load(key string) (value *Value, ok bool) {
  raw, has := c.m.Load(key)
  if !has {
    return nil, false
  }
  value, ok = raw.(*Value)
  return
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *Collection) LoadOrStore(key string, value interface{}) (actual *Value, loaded bool) {
  val, _ := NewValue(value)
  raw, has := c.m.LoadOrStore(key, &val)
  if has  {
    actual, loaded = raw.(*Value)
  }
	return
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (c *Collection) LoadAndDelete(key string) (value *Value, loaded bool) {
  raw, loaded := c.m.LoadAndDelete(key)
  value, _ = raw.(*Value)
	return
}

// Delete deletes the value for a key.
func (c *Collection) Delete(key string) {
	c.LoadAndDelete(key)
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
func (c *Collection) Range(f func(key string, value *Value) bool) {
  c.m.Range(func(k, v interface{}) bool {
		return f(k.(string), v.(*Value))
	})
}
