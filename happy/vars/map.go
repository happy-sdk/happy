// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars

import (
	"sync"
	"sync/atomic"
)

// Collection is collection of Variables safe for concurrent use.
type Map struct {
	mu  sync.RWMutex
	len int64
	db  map[string]Variable
}

// Store sets the value for a key.
// Error is returned when key or value parsing fails
// or variable is already set and is readonly.
func (c *Map) Store(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db == nil {
		c.db = make(map[string]Variable)
	}

	curr, ok := c.db[key]
	if ok && curr.ReadOnly() {
		return errorf("%w: can not set value for %s", ErrReadOnly, key)
	}

	if v, ok := value.(Variable); ok && v.Name() == key {
		c.db[key] = v
		atomic.AddInt64(&c.len, 1)
		return nil
	}

	v, err := New(key, value, false)
	if err != nil {
		return err
	}
	c.db[key] = v
	atomic.AddInt64(&c.len, 1)
	return err
}

func (c *Map) StoreReadOnly(key string, value any, ro bool) error {
	v, err := New(key, value, ro)
	if err != nil {
		return err
	}
	return c.Store(key, v)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c *Map) Get(key string) (v Variable) {
	v, ok := c.db[key]
	if !ok {
		return EmptyVariable
	}
	return v
}

// Has reprts whether given variable  exists.
func (c *Map) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.db[key]
	return ok
}

func (c *Map) All() (all []Variable) {
	c.Range(func(v Variable) bool {
		all = append(all, v)
		return true
	})
	return
}

// Delete deletes the value for a key.
func (c *Map) Delete(key string) {
	_, _ = c.LoadAndDelete(key)
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (c *Map) Load(key string) (v Variable, ok bool) {
	if !c.Has(key) {
		return EmptyVariable, false
	}
	return c.Get(key), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (c *Map) LoadAndDelete(key string) (v Variable, loaded bool) {
	if !c.Has(key) {
		return EmptyVariable, false
	}

	v = c.Get(key)
	loaded = true
	c.mu.Lock()
	delete(c.db, v.Name())
	atomic.AddInt64(&c.len, -1)
	c.mu.Unlock()
	return
}

// LoadOrDefault returns the existing value for the key if present.
// Much like LoadOrStore, but second argument willl be returned as
// Value whithout being stored into Map.
func (c *Map) LoadOrDefault(key string, value any) (v Variable, loaded bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(key) > 0 {
		if def, ok := value.(Variable); ok {
			return def, false
		}
	}
	// existing
	if val, ok := c.db[key]; ok {
		return val, true
	}

	v, err := New(key, value, false)
	if err != nil {
		return EmptyVariable, false
	}
	return v, false
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *Map) LoadOrStore(key string, value any) (actual Variable, loaded bool) {
	k, err := parseKey(key)
	if err != nil {
		return EmptyVariable, false
	}
	loaded = c.Has(k)
	if !loaded {
		c.Store(k, value)
	}
	return c.Get(k), loaded
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
func (c *Map) Range(f func(v Variable) bool) {
	c.mu.RLock()
	for _, v := range c.db {
		c.mu.RUnlock()
		if !f(v) {
			break
		}
		c.mu.RLock()
	}
	c.mu.RUnlock()
}

// ToBytes returns []byte containing
// key = "value"\n.
func (c *Map) ToBytes() []byte {
	s := c.ToKeyValSlice()

	p := getParser()
	defer p.free()

	for _, line := range s {
		p.fmt.string(line + "\n")
	}
	return p.buf
}

// ToKeyValSlice produces []string slice of strings in format key = "value".
func (c *Map) ToKeyValSlice() []string {
	r := []string{}
	c.Range(func(v Variable) bool {
		// we can do it directly on interface value since they all are Values
		// implementing Stringer
		r = append(r, v.Name()+"=\""+v.String()+"\"")
		return true
	})
	return r
}

// Len of collection.
func (c *Map) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int(atomic.LoadInt64(&c.len))
}

// GetWithPrefix return all variables with prefix if any as new Map
// and strip prefix from keys.
func (c *Map) ExtractWithPrefix(prfx string) *Map {
	vars := new(Map)
	c.Range(func(v Variable) bool {
		key := v.Name()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			_ = vars.Store(key[len(prfx):], v)
		}
		return true
	})
	return vars
}

// LoadWithPrefix return all variables with prefix if any as new Map.
func (c *Map) LoadWithPrefix(prfx string) (set *Map, loaded bool) {
	set = new(Map)
	c.Range(func(v Variable) bool {
		key := v.Name()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			_ = set.Store(key, v)
			loaded = true
		}
		return true
	})
	return set, loaded
}
