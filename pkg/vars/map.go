// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars

import (
	"encoding/json"
	"fmt"
	"iter"
	"sort"
	"sync"
	"sync/atomic"
)

// Map is collection of Variables safe for concurrent use.
type Map struct {
	mu  sync.RWMutex
	len int64
	db  map[string]Variable
}

// NewMap creates a new Map instance.
func NewMap() *Map {
	return &Map{
		db: make(map[string]Variable),
	}
}

// Store sets the value for a key.
// Error is returned when key or value parsing fails
// or variable is already set and is readonly.
func (m *Map) Store(key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db == nil {
		m.db = make(map[string]Variable)
	}

	curr, has := m.db[key]
	if has && curr.ReadOnly() {
		return errorf("%w: can not set value for %s", ErrReadOnly, key)
	}

	if v, ok := value.(Variable); ok && v.Name() == key {
		m.db[key] = v
		if !has {
			atomic.AddInt64(&m.len, 1)
		}
		return nil
	}

	v, err := New(key, value, false)
	if err != nil {
		return err
	}
	m.db[key] = v
	if !has {
		atomic.AddInt64(&m.len, 1)
	}
	return err
}

func (m *Map) StoreReadOnly(key string, value any, ro bool) error {
	v, err := New(key, value, ro)
	if err != nil {
		return err
	}
	return m.Store(key, v)
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (m *Map) Get(key string) (v Variable) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.db[key]
	if !ok {
		return EmptyVariable
	}
	return v
}

// Has reprts whether given variable  exists.
func (m *Map) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.db[key]
	return ok
}

// All returns an iterator that yields all variables in the map.
// The iteration order is deterministic, based on sorted keys.
//
// All creates a snapshot of the map at the time it's called, so concurrent
// modifications to the map during iteration will not affect the yielded values.
// This ensures that nested iterations or concurrent map operations don't
// interfere with each other.
//
// The returned iterator can be used with Go's range-over-func feature:
//
//	for v := range m.All() {
//		fmt.Printf("%s = %s\n", v.Name(), v.String())
//	}
//
// If the iterator's yield function returns false, iteration stops early.
func (m *Map) All() iter.Seq[Variable] {
	return func(yield func(Variable) bool) {
		m.mu.RLock()
		// Create a snapshot of all variables at this moment
		snapshot := make([]Variable, 0, len(m.db))
		keys := make([]string, 0, len(m.db))

		for key := range m.db {
			keys = append(keys, key)
		}

		// Sort keys for consistent iteration order
		sort.Strings(keys)

		// Build snapshot with current values
		for _, key := range keys {
			if v, exists := m.db[key]; exists {
				snapshot = append(snapshot, v)
			}
		}
		m.mu.RUnlock()

		// Iterate over snapshot without holding locks
		for _, v := range snapshot {
			if !yield(v) {
				return
			}
		}
	}
}

// Delete deletes the value for a key.
func (m *Map) Delete(key string) {
	_, _ = m.LoadAndDelete(key)
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (m *Map) Load(key string) (v Variable, ok bool) {
	if !m.Has(key) {
		return EmptyVariable, false
	}
	return m.Get(key), true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *Map) LoadAndDelete(key string) (v Variable, loaded bool) {
	if !m.Has(key) {
		return EmptyVariable, false
	}

	v = m.Get(key)
	loaded = true
	m.mu.Lock()
	delete(m.db, v.Name())
	atomic.AddInt64(&m.len, -1)
	m.mu.Unlock()
	return
}

// LoadOrDefault returns the existing value for the key if present.
// Much like LoadOrStore, but second argument willl be returned as
// Value whithout being stored into Map.
func (m *Map) LoadOrDefault(key string, value any) (v Variable, loaded bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(key) > 0 {
		if def, ok := value.(Variable); ok {
			return def, false
		}
	}
	// existing
	if val, ok := m.db[key]; ok {
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
func (m *Map) LoadOrStore(key string, value any) (actual Variable, loaded bool) {
	k, err := parseKey(key)
	if err != nil {
		return EmptyVariable, false
	}
	loaded = m.Has(k)
	if !loaded {
		// we can't really handle that error here
		_ = m.Store(k, value)
	}
	return m.Get(k), loaded
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
func (m *Map) Range(f func(v Variable) bool) {
	m.mu.RLock()
	// Create a snapshot of all variables at this moment
	snapshot := make([]Variable, 0, len(m.db))
	keys := make([]string, 0, len(m.db))

	for key := range m.db {
		keys = append(keys, key)
	}

	// Sort keys for consistent iteration order
	sort.Strings(keys)

	// Build snapshot with current values
	for _, key := range keys {
		if v, exists := m.db[key]; exists {
			snapshot = append(snapshot, v)
		}
	}
	m.mu.RUnlock()

	// Iterate over snapshot without holding locks
	for _, v := range snapshot {
		if !f(v) {
			break
		}
	}
}

// ToBytes returns []byte containing
// key = "value"\n.
func (m *Map) ToBytes() []byte {
	s := m.ToKeyValSlice()

	p := getParser()
	defer p.free()

	for _, line := range s {
		p.fmt.string(line + "\n")
	}
	return p.buf
}

// ToKeyValSlice produces []string slice of strings in format key = "value".
func (m *Map) ToKeyValSlice() []string {
	r := []string{}
	m.Range(func(v Variable) bool {
		// we can do it directly on interface value since they all are Values
		// implementing Stringer
		r = append(r, v.Name()+"="+v.String())
		return true
	})
	return r
}

// Len of collection.
func (m *Map) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int(atomic.LoadInt64(&m.len))
}

// GetWithPrefix return all variables with prefix if any as new Map
// and strip prefix from keys.
func (m *Map) ExtractWithPrefix(prfx string) *Map {
	vars := new(Map)
	m.Range(func(v Variable) bool {
		key := v.Name()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			_ = vars.Store(key[len(prfx):], v)
		}
		return true
	})
	return vars
}

// LoadWithPrefix return all variables with prefix if any as new Map.
func (m *Map) LoadWithPrefix(prfx string) (set *Map, loaded bool) {
	set = new(Map)
	m.Range(func(v Variable) bool {
		key := v.Name()
		if len(key) >= len(prfx) && key[0:len(prfx)] == prfx {
			_ = set.Store(key, v)
			loaded = true
		}
		return true
	})
	return set, loaded
}

func (m *Map) MarshalJSON() ([]byte, error) {
	// Create a map to hold the key-value pairs of the synm.Map
	var objMap = make(map[string]any)

	// Iterate over the synm.Map and add the key-value pairs to the map
	m.Range(func(v Variable) bool {
		objMap[v.Name()] = v.Any()
		return true
	})

	// Use json.Marshal to convert the map to JSON
	return json.Marshal(objMap)
}

func (m *Map) UnmarshalJSON(data []byte) error {
	// Create a map to hold the key-value pairs from the JSON data
	var objMap map[string]any

	// Use json.Unmarshal to parse the JSON data into the map
	if err := json.Unmarshal(data, &objMap); err != nil {
		return err
	}

	// Iterate over the map and add the key-value pairs to the synm.Map
	for key, value := range objMap {
		if err := m.Store(key, value); err != nil {
			return err
		}
	}

	return nil
}

func (m *Map) ToGoMap() map[string]string {
	mm := make(map[string]string)
	m.Range(func(v Variable) bool {
		mm[v.Name()] = v.String()
		return true
	})
	return mm
}

// Collection is collection of Variables safe for concurrent use.
type ReadOnlyMap struct {
	mu  sync.RWMutex
	len int64
	db  map[string]Variable
}

func ReadOnlyMapFrom(m *Map) (rom *ReadOnlyMap, err error) {
	rom = new(ReadOnlyMap)
	m.Range(func(v Variable) bool {
		err = rom.storeReadOnly(v.Name(), v, true)
		return err == nil
	})
	return
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the variable is not set
// or value was empty.
func (m *ReadOnlyMap) Get(key string) (v Variable) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.db[key]
	if !ok {
		return EmptyVariable
	}
	return v
}

// Has reprts whether given variable  exists.
func (m *ReadOnlyMap) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.db[key]
	return ok
}

func (m *ReadOnlyMap) All() (all []Variable) {
	m.Range(func(v Variable) bool {
		all = append(all, v)
		return true
	})
	return
}

// Load returns the variable stored in the Collection for a key,
// or EmptyVar if no value is present.
// The ok result indicates whether variable was found in the Collection.
func (m *ReadOnlyMap) Load(key string) (v Variable, ok bool) {
	if !m.Has(key) {
		return EmptyVariable, false
	}
	return m.Get(key), true
}

// LoadOrDefault returns the existing value for the key if present.
// Much like LoadOrStore, but second argument willl be returned as
// Value whithout being stored into Map.
func (m *ReadOnlyMap) LoadOrDefault(key string, value any) (v Variable, loaded bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(key) > 0 {
		if def, ok := value.(Variable); ok {
			return def, false
		}
	}
	// existing
	if val, ok := m.db[key]; ok {
		return val, true
	}

	v, err := New(key, value, false)
	if err != nil {
		return EmptyVariable, false
	}
	return v, false
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
func (m *ReadOnlyMap) Range(f func(v Variable) bool) {
	m.mu.RLock()
	keys := make([]string, len(m.db))
	i := 0
	for key := range m.db {
		keys[i] = key
		i++
	}
	m.mu.RUnlock()

	sort.Strings(keys)

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, key := range keys {
		v := m.db[key]
		if !f(v) {
			break
		}
	}
}

// ToBytes returns []byte containing
// key = "value"\n.
func (m *ReadOnlyMap) ToBytes() []byte {
	s := m.ToKeyValSlice()

	p := getParser()
	defer p.free()

	for _, line := range s {
		p.fmt.string(line + "\n")
	}
	return p.buf
}

// ToKeyValSlice produces []string slice of strings in format key = "value".
func (m *ReadOnlyMap) ToKeyValSlice() []string {
	r := []string{}
	m.Range(func(v Variable) bool {
		// we can do it directly on interface value since they all are Values
		// implementing Stringer
		r = append(r, v.Name()+"="+v.String())
		return true
	})
	return r
}

// Len of collection.
func (m *ReadOnlyMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int(atomic.LoadInt64(&m.len))
}

// GetWithPrefix return all variables with prefix if any as new Map
// and strip prefix from keys.
func (m *ReadOnlyMap) WithPrefix(prefix string) (rmap *ReadOnlyMap, err error) {
	rmap = new(ReadOnlyMap)
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range m.db {
		key := v.Name()

		if len(key) >= len(prefix) && key[0:len(prefix)] == prefix {
			err = rmap.storeReadOnly(key[len(prefix):], v, true)
			if err != nil {
				return
			}
		}
	}
	return
}

// LoadWithPrefix return all variables with prefix if any as new Map.
func (m *ReadOnlyMap) LoadWithPrefix(prefix string) (set *ReadOnlyMap, loaded bool) {
	set, _ = m.WithPrefix(prefix)
	loaded = set.Len() > 0
	return
}

func (m *ReadOnlyMap) MarshalJSON() ([]byte, error) {
	// Create a map to hold the key-value pairs of the synm.Map
	var objMap = make(map[string]any)

	// Iterate over the synm.Map and add the key-value pairs to the map
	m.Range(func(v Variable) bool {
		objMap[v.Name()] = v.Any()
		return true
	})

	if len(objMap) == 0 {
		return []byte("null"), nil
	}

	// Use json.Marshal to convert the map to JSON
	b, err := json.Marshal(objMap)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (m *ReadOnlyMap) UnmarshalJSON(data []byte) error {
	// Create a map to hold the key-value pairs from the JSON data
	var objMap map[string]any

	// Use json.Unmarshal to parse the JSON data into the map
	if err := json.Unmarshal(data, &objMap); err != nil {
		return err
	}

	// Iterate over the map and add the key-value pairs to the synm.Map
	for key, value := range objMap {
		if err := m.storeReadOnly(key, value, true); err != nil {
			return err
		}
	}

	return nil
}

// Store sets the value for a key.
// Error is returned when key or value parsing fails
// or variable is already set and is readonly.
func (m *ReadOnlyMap) store(key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db == nil {
		m.db = make(map[string]Variable)
	}

	curr, has := m.db[key]
	if has && curr.ReadOnly() {
		return errorf("%w: can not set value for %s", ErrReadOnly, key)
	}

	if v, ok := value.(Variable); ok && v.Name() == key {
		m.db[key] = v
		if !has {
			atomic.AddInt64(&m.len, 1)
		}
		return nil
	}
	return fmt.Errorf("%w: %s", Error, key)
}

func (m *ReadOnlyMap) storeReadOnly(key string, value any, ro bool) error {
	v, err := New(key, value, ro)
	if err != nil {
		return err
	}
	return m.store(key, v)
}
