// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vars

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"
	"unsafe"
)

// Set updates key value pair in collection. If key does not exist then appends
// key wth given value
func (c *Collection) Set(k string, v interface{}) {
	k = strings.TrimSpace(k)
	vv, _ := New(k, v)
	c.Store(vv)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (c *Collection) Get(k string, defval ...interface{}) (val Value) {
	value, ok := c.Load(k)
	if len(k) == 0 || !ok || value.Len() == 0 {
		if len(defval) > 0 {
			s, _ := NewValue(defval[0])
			return s
		}
	}
	return value.val
}

// Has reprts whether given variable  exists
func (c *Collection) Has(k string) bool {
	_, ok := c.Load(k)
	return ok
}

// Len of collection
func (c *Collection) Len() int {
	return int(c.len)
}

// GetWithPrefix return all variables with prefix if any as map[]
func (c *Collection) GetWithPrefix(prfx string) (vars Collection) {
	vars = Collection{}
	c.Range(func(key string, value Variable) bool {
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			vars.Store(value)
		}
		return true
	})
	return
}

// ToKeyValSlice produces []string slice of strings in format key = "value"
func (c *Collection) ToKeyValSlice() []string {
	r := []string{}
	c.Range(func(key string, value Variable) bool {
		r = append(r, fmt.Sprintf("%s = %q", key, value.String()))
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
func (c *Collection) Store(variable Variable) {
	read, _ := c.read.Load().(readOnly)
	if e, ok := read.m[variable.Key()]; ok && e.tryStore(&variable) {
		return
	}

	c.mu.Lock()
	read, _ = c.read.Load().(readOnly)
	if e, ok := read.m[variable.Key()]; ok {
		if e.unexpungeLocked() {
			// The entry was previously expunged, which implies that there is a
			// non-nil dirty map and this entry is not in it.
			c.dirty[variable.Key()] = e
		}
		e.storeLocked(&variable)
	} else if e, ok := c.dirty[variable.Key()]; ok {
		e.storeLocked(&variable)
	} else {
		if !read.amended {
			// We're adding the first new key to the dirty map.
			// Make sure it is allocated and mark the read-only map as incomplete.
			c.dirtyLocked()
			c.read.Store(readOnly{m: read.m, amended: true})
		}
		c.dirty[variable.Key()] = newEntry(variable)
	}
	atomic.AddInt64(&c.len, 1)
	c.mu.Unlock()
}

func newEntry(i Variable) *entry {
	return &entry{p: unsafe.Pointer(&i)}
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (c *Collection) Load(key string) (variable Variable, ok bool) {
	read, _ := c.read.Load().(readOnly)
	e, ok := read.m[key]
	if !ok && read.amended {
		c.mu.Lock()
		// Avoid reporting a spurious miss if m.dirty got promoted while we were
		// blocked on m.mu. (If further loads of the same key will not miss, it's
		// not worth copying the dirty map for this key.)
		read, _ = c.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = c.dirty[key]
			// Regardless of whether the entry was present, record a miss: this key
			// will take the slow path until the dirty map is promoted to the read
			// map.
			c.missLocked()
		}
		c.mu.Unlock()
	}
	if !ok {
		return EmptyVar, false
	}
	return e.load()
}

func (e *entry) load() (value Variable, ok bool) {
	p := atomic.LoadPointer(&e.p)
	if p == nil || p == expunged {
		return EmptyVar, false
	}
	return *(*Variable)(p), true
}

// tryStore stores a value if the entry has not been expunged.
//
// If the entry is expunged, tryStore returns false and leaves the entry
// unchanged.
func (e *entry) tryStore(i *Variable) bool {
	for {
		p := atomic.LoadPointer(&e.p)
		if p == expunged {
			return false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, unsafe.Pointer(i)) {
			return true
		}
	}
}

// unexpungeLocked ensures that the entry is not marked as expunged.
//
// If the entry was previously expunged, it must be added to the dirty map
// before m.mu is unlocked.
func (e *entry) unexpungeLocked() (wasExpunged bool) {
	return atomic.CompareAndSwapPointer(&e.p, expunged, nil)
}

// storeLocked unconditionally stores a value to the entry.
//
// The entry must be known not to be expunged.
func (e *entry) storeLocked(i *Variable) {
	atomic.StorePointer(&e.p, unsafe.Pointer(i))
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (c *Collection) LoadOrStore(variable Variable) (actual Variable, loaded bool) {
	// Avoid locking if it's a clean hit.
	read, _ := c.read.Load().(readOnly)
	if e, ok := read.m[variable.Key()]; ok {
		actual, loaded, ok := e.tryLoadOrStore(variable)
		if ok {
			return actual, loaded
		}
	}

	c.mu.Lock()
	read, _ = c.read.Load().(readOnly)
	if e, ok := read.m[variable.Key()]; ok {
		if e.unexpungeLocked() {
			c.dirty[variable.Key()] = e
		}
		actual, loaded, _ = e.tryLoadOrStore(variable)
	} else if e, ok := c.dirty[variable.Key()]; ok {
		actual, loaded, _ = e.tryLoadOrStore(variable)
		c.missLocked()
	} else {
		if !read.amended {
			// We're adding the first new key to the dirty map.
			// Make sure it is allocated and mark the read-only map as incomplete.
			c.dirtyLocked()
			c.read.Store(readOnly{m: read.m, amended: true})
		}
		c.dirty[variable.Key()] = newEntry(variable)
		actual, loaded = variable, false
	}
	c.mu.Unlock()

	return actual, loaded
}

// tryLoadOrStore atomically loads or stores a value if the entry is not
// expunged.
//
// If the entry is expunged, tryLoadOrStore leaves the entry unchanged and
// returns with ok==false.
func (e *entry) tryLoadOrStore(i Variable) (actual Variable, loaded, ok bool) {
	p := atomic.LoadPointer(&e.p)
	if p == expunged {
		return EmptyVar, false, false
	}
	if p != nil {
		return *(*Variable)(p), true, true
	}

	// Copy the interface after the first load to make this method more amenable
	// to escape analysis: if we hit the "load" path or the entry is expunged, we
	// shouldn't bother heap-allocating.
	ic := i
	for {
		if atomic.CompareAndSwapPointer(&e.p, nil, unsafe.Pointer(&ic)) {
			return i, false, true
		}
		p = atomic.LoadPointer(&e.p)
		if p == expunged {
			return EmptyVar, false, false
		}
		if p != nil {
			return *(*Variable)(p), true, true
		}
	}
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (c *Collection) LoadAndDelete(key string) (value Variable, loaded bool) {
	read, _ := c.read.Load().(readOnly)
	e, ok := read.m[key]
	if !ok && read.amended {
		c.mu.Lock()
		read, _ = c.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = c.dirty[key]
			delete(c.dirty, key)
			// Regardless of whether the entry was present, record a miss: this key
			// will take the slow path until the dirty map is promoted to the read
			// map.
			c.missLocked()
		}
		c.mu.Unlock()
	}
	if ok {
		return e.delete()
	}
	return EmptyVar, false
}

// Delete deletes the value for a key.
func (c *Collection) Delete(key string) {
	c.LoadAndDelete(key)
}

func (e *entry) delete() (value Variable, ok bool) {
	for {
		p := atomic.LoadPointer(&e.p)
		if p == nil || p == expunged {
			return EmptyVar, false
		}
		if atomic.CompareAndSwapPointer(&e.p, p, nil) {
			return *(*Variable)(p), true
		}
	}
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
func (c *Collection) Range(f func(key string, value Variable) bool) {
	// We need to be able to iterate over all of the keys that were already
	// present at the start of the call to Range.
	// If read.amended is false, then read.m satisfies that property without
	// requiring us to hold m.mu for a long time.
	read, _ := c.read.Load().(readOnly)
	if read.amended {
		// m.dirty contains keys not in read.m. Fortunately, Range is already O(N)
		// (assuming the caller does not break out early), so a call to Range
		// amortizes an entire copy of the map: we can promote the dirty copy
		// immediately!
		c.mu.Lock()
		read, _ = c.read.Load().(readOnly)
		if read.amended {
			read = readOnly{m: c.dirty}
			c.read.Store(read)
			c.dirty = nil
			c.misses = 0
		}
		c.mu.Unlock()
	}

	for k, e := range read.m {
		v, ok := e.load()
		if !ok {
			continue
		}
		if !f(k, v) {
			break
		}
	}
}

func (c *Collection) missLocked() {
	c.misses++
	if c.misses < len(c.dirty) {
		return
	}
	c.read.Store(readOnly{m: c.dirty})
	c.dirty = nil
	c.misses = 0
}

func (c *Collection) dirtyLocked() {
	if c.dirty != nil {
		return
	}

	read, _ := c.read.Load().(readOnly)
	c.dirty = make(map[string]*entry, len(read.m))
	for k, e := range read.m {
		if !e.tryExpungeLocked() {
			c.dirty[k] = e
		}
	}
}

func (e *entry) tryExpungeLocked() (isExpunged bool) {
	p := atomic.LoadPointer(&e.p)
	for p == nil {
		if atomic.CompareAndSwapPointer(&e.p, nil, expunged) {
			return true
		}
		p = atomic.LoadPointer(&e.p)
	}
	return p == expunged
}
